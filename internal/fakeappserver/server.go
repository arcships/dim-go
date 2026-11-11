package fakeappserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/transport"
)

// socketFile 与真实客户端的端点规则一致：<dataDir>/rust/daemon.sock。
const socketFile = "daemon.sock"

// maxSocketPathBytes 是保守的 Unix socket 路径上限；macOS 为 104 字节。
const maxSocketPathBytes = 100

// Direction 是帧的方向。
type Direction string

const (
	// DirectionReceived 表示客户端发来的帧。
	DirectionReceived Direction = "received"
	// DirectionSent 表示假服务端发出的帧。
	DirectionSent Direction = "sent"
)

// Frame 是一条已收发的原始帧。收到的每一帧都记录，包括坏帧与客户端通知。
type Frame struct {
	At        time.Time
	Conn      int
	Direction Direction
	// Method 对请求、通知与回执都是方法名；无法识别时为空。
	Method string
	// ID 是请求/回执的标识 wire 原始文本；通知为空。
	ID  protocol.ID
	Raw []byte
}

// Request 是一条已解析的请求帧。
type Request struct {
	At         time.Time
	Conn       int
	ID         protocol.ID
	Method     string
	Params     json.RawMessage
	Occurrence int
}

// serverConn 是一条客户端连接的状态。
type serverConn struct {
	id        int
	conn      net.Conn
	writeMu   sync.Mutex
	closeOnce sync.Once
	// attached 由 Server.mu 保护：记录这条连接 attach 过的会话。
	attached map[string]bool
}

// sessionState 是一个会话的可变事实。Server.mu 保护读写。
type sessionState struct {
	spec Session
	head int64
}

// Server 是一个按脚本应答的假 App Server。所有导出方法可并发调用。
type Server struct {
	script     Script
	dataDir    string
	endpoint   transport.Endpoint
	listener   net.Listener
	ownDataDir bool

	mu         sync.Mutex
	closed     bool
	closeOnce  sync.Once
	nextConnID int
	conns      map[*serverConn]struct{}
	accepted   int
	sessions   map[string]*sessionState
	frames     []Frame
	requests   []Request
	// methodCounts 记录每个方法收到的请求次数，用于按出现次数命中故障。
	methodCounts  map[string]int
	replyByMethod map[string]Reply
	faults        []Fault
}

// Start 在临时数据目录里启动假服务端；Close 时连同目录一起删除。
//
// 三行用法：
//
//	server, err := fakeappserver.Start(fakeappserver.Script{Sessions: sessions})
//	defer server.Close()
//	supervisor, err := daemon.New(daemon.Config{DataDir: server.DataDir()})
func Start(script Script) (*Server, error) {
	// Unix socket 路径有平台上限（macOS 104 字节、Linux 108 字节），临时目录
	// 优先用短路径的系统临时目录以留出余量；失败时回落到默认临时目录。
	dir, err := os.MkdirTemp("/tmp", "dimagent-fakeappserver-")
	if err != nil {
		dir, err = os.MkdirTemp("", "dimagent-fakeappserver-")
	}
	if err != nil {
		return nil, fmt.Errorf("创建假服务端临时目录: %w", err)
	}
	server, err := StartIn(dir, script)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	server.ownDataDir = true
	return server, nil
}

// StartIn 在指定数据目录下启动假服务端，端点与真实客户端解析出的路径一致。
// 调用方负责该目录的生命周期；Close 只删除 socket 文件，不删目录。
func StartIn(dataDir string, script Script) (*Server, error) {
	if runtime.GOOS == "windows" {
		return nil, errors.New("假服务端只支持 Unix socket；Windows 命名管道未实现")
	}
	if dataDir == "" {
		return nil, errors.New("假服务端缺少数据目录")
	}
	script = script.withDefaults()
	if err := validateScript(script); err != nil {
		return nil, err
	}
	home := filepath.Join(dataDir, "rust")
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, fmt.Errorf("准备假服务端 home: %w", err)
	}
	address := filepath.Join(home, socketFile)
	// Unix socket 路径长度有平台上限（macOS 104、Linux 108）；超过时直接报错，
	// 而不是让调用方看到一个难懂的 connect: invalid argument。
	if len(address) > maxSocketPathBytes {
		return nil, fmt.Errorf(
			"socket 路径 %d 字节，超过平台上限 %d；请把数据目录换到更短的路径",
			len(address), maxSocketPathBytes,
		)
	}
	// 上一次异常退出可能留下 socket 文件；先清掉再监听。
	if err := os.Remove(address); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("清理旧 socket: %w", err)
	}
	listener, err := net.Listen("unix", address)
	if err != nil {
		return nil, fmt.Errorf("监听 socket %s: %w", address, err)
	}

	server := &Server{
		script:        script,
		dataDir:       dataDir,
		endpoint:      transport.Endpoint{Network: transport.NetworkUnix, Address: address},
		listener:      listener,
		conns:         make(map[*serverConn]struct{}),
		sessions:      make(map[string]*sessionState, len(script.Sessions)),
		methodCounts:  make(map[string]int),
		replyByMethod: make(map[string]Reply, len(script.Replies)),
		faults:        append([]Fault(nil), script.Faults...),
	}
	for _, reply := range script.Replies {
		server.replyByMethod[reply.Method] = reply
	}
	for _, spec := range script.Sessions {
		server.sessions[spec.ID] = newSessionState(spec)
	}
	go server.accept()
	return server, nil
}

func validateScript(script Script) error {
	seen := make(map[string]struct{}, len(script.Sessions))
	for _, spec := range script.Sessions {
		if spec.ID == "" {
			return errors.New("会话缺少标识")
		}
		if _, duplicate := seen[spec.ID]; duplicate {
			return fmt.Errorf("会话标识重复: %q", spec.ID)
		}
		seen[spec.ID] = struct{}{}
	}
	for _, reply := range script.Replies {
		if reply.Method == "" {
			return errors.New("脚本应答缺少方法名")
		}
		if reply.Result != nil && reply.Error != nil {
			return fmt.Errorf("方法 %s 的脚本应答不能同时给 result 与 error", reply.Method)
		}
	}
	for _, fault := range script.Faults {
		if fault.Method == "" {
			return errors.New("故障注入缺少方法名")
		}
		if fault.Occurrence < 0 {
			return fmt.Errorf("方法 %s 的故障出现次数不能为负", fault.Method)
		}
	}
	return nil
}

func newSessionState(spec Session) *sessionState {
	if spec.BranchID == "" {
		spec.BranchID = "main"
	}
	spec.Events = append([]protocol.RawEvent(nil), spec.Events...)
	spec.AttachReplay = append([]protocol.RawEvent(nil), spec.AttachReplay...)
	head := spec.Head
	for _, event := range spec.Events {
		if event.Sequence > head {
			head = event.Sequence
		}
	}
	return &sessionState{spec: spec, head: head}
}

// DataDir 返回假服务端的数据目录；可直接交给 daemon.Config。
func (s *Server) DataDir() string { return s.dataDir }

// Endpoint 返回本地端点；可直接交给 transport.Dial。
func (s *Server) Endpoint() transport.Endpoint { return s.endpoint }

// InstanceID 返回 hello 回执里的实例标识。
func (s *Server) InstanceID() string { return s.script.InstanceID }

// Accepted 返回累计接受的连接数。
func (s *Server) Accepted() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accepted
}

// Connections 返回当前仍在跟踪的连接数。
func (s *Server) Connections() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.conns)
}

// Frames 返回全部收发帧的副本，按发生顺序。
func (s *Server) Frames() []Frame {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneFrames(s.frames)
}

// ReceivedFrames 只返回客户端发来的帧副本。
func (s *Server) ReceivedFrames() []Frame {
	s.mu.Lock()
	defer s.mu.Unlock()
	received := make([]Frame, 0, len(s.frames))
	for _, frame := range s.frames {
		if frame.Direction == DirectionReceived {
			received = append(received, cloneFrame(frame))
		}
	}
	return received
}

// Requests 返回全部已解析请求的副本，按到达顺序。
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneRequests(s.requests)
}

// RequestsFor 返回指定方法的请求副本，按到达顺序。
func (s *Server) RequestsFor(method string) []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	matched := make([]Request, 0, len(s.requests))
	for _, request := range s.requests {
		if request.Method == method {
			matched = append(matched, cloneRequest(request))
		}
	}
	return matched
}

// Methods 返回收到过的方法名序列（含重复），按到达顺序，便于核对请求次数。
func (s *Server) Methods() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	methods := make([]string, 0, len(s.requests))
	for _, request := range s.requests {
		methods = append(methods, request.Method)
	}
	return methods
}

// AttachedSessions 返回当前 attach 过的会话标识，按标识排序。
func (s *Server) AttachedSessions() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	set := make(map[string]struct{})
	for conn := range s.conns {
		for sessionID := range conn.attached {
			set[sessionID] = struct{}{}
		}
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Notify 向所有连接推送一条通知；params 必须可 JSON 编码。
func (s *Server) Notify(method string, params any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("编码通知 %s: %w", method, err)
	}
	return s.NotifyRaw(method, raw)
}

// NotifyRaw 向所有连接推送一条通知，params 已经是 wire 形状的 JSON。
func (s *Server) NotifyRaw(method string, params json.RawMessage) error {
	frame, err := notificationFrame(method, params)
	if err != nil {
		return err
	}
	s.mu.Lock()
	conns := make([]*serverConn, 0, len(s.conns))
	for conn := range s.conns {
		conns = append(conns, conn)
	}
	s.mu.Unlock()
	for _, conn := range conns {
		_ = conn.write(s, frame, method, "")
	}
	return nil
}

// PushEvent 把一条事实追加到会话历史，并向已 attach 该会话的连接推送
// session.event。重复调用同一 event 即可注入重复通知。
func (s *Server) PushEvent(sessionID string, event protocol.RawEvent) error {
	return s.PushEventTimes(sessionID, event, 1)
}

// PushEventTimes 把一份事实追加到历史后连发 times 次通知，用于注入重复通知。
func (s *Server) PushEventTimes(sessionID string, event protocol.RawEvent, times int) error {
	if times <= 0 {
		return nil
	}
	s.mu.Lock()
	state, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("未知会话 %q", sessionID)
	}
	state.spec.Events = append(state.spec.Events, event)
	if event.Sequence > state.head {
		state.head = event.Sequence
	}
	branchID := state.spec.BranchID
	conns := make([]*serverConn, 0, len(s.conns))
	for conn := range s.conns {
		if conn.attached[sessionID] {
			conns = append(conns, conn)
		}
	}
	s.mu.Unlock()

	params, err := json.Marshal(protocol.SessionEventParams{
		SessionID: sessionID,
		BranchID:  branchID,
		Event:     event,
	})
	if err != nil {
		return fmt.Errorf("编码 session.event: %w", err)
	}
	frame, err := notificationFrame("session.event", params)
	if err != nil {
		return err
	}
	for i := 0; i < times; i++ {
		for _, conn := range conns {
			_ = conn.write(s, frame, "session.event", "")
		}
	}
	return nil
}

// AppendEvent 只把事实追加到会话历史并推进头部，不推送通知；用于构造
// 断线期间服务端已经发生、需要重读才能拿到的事实。
func (s *Server) AppendEvent(sessionID string, event protocol.RawEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("未知会话 %q", sessionID)
	}
	state.spec.Events = append(state.spec.Events, event)
	if event.Sequence > state.head {
		state.head = event.Sequence
	}
	return nil
}

// Head 返回会话当前头部序号。
func (s *Server) Head(sessionID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.sessions[sessionID]
	if !ok {
		return 0, fmt.Errorf("未知会话 %q", sessionID)
	}
	return state.head, nil
}

// DisconnectAll 断开全部连接但继续监听，返回断开的连接数。
func (s *Server) DisconnectAll() int {
	s.mu.Lock()
	targets := make([]*serverConn, 0, len(s.conns))
	for conn := range s.conns {
		targets = append(targets, conn)
	}
	s.mu.Unlock()
	for _, conn := range targets {
		s.closeConn(conn)
	}
	return len(targets)
}

// DisconnectStreams 只断开 attach 过会话的连接，返回断开的连接数。
// 控制连接不受影响，用于验证会话流断线重连。
func (s *Server) DisconnectStreams() int {
	s.mu.Lock()
	targets := make([]*serverConn, 0)
	for conn := range s.conns {
		if len(conn.attached) > 0 {
			targets = append(targets, conn)
		}
	}
	s.mu.Unlock()
	for _, conn := range targets {
		s.closeConn(conn)
	}
	return len(targets)
}

// DisconnectSession 断开 attach 过指定会话的连接，返回断开的连接数。
func (s *Server) DisconnectSession(sessionID string) int {
	s.mu.Lock()
	targets := make([]*serverConn, 0)
	for conn := range s.conns {
		if conn.attached[sessionID] {
			targets = append(targets, conn)
		}
	}
	s.mu.Unlock()
	for _, conn := range targets {
		s.closeConn(conn)
	}
	return len(targets)
}

// Close 停止监听、断开全部连接并清理 socket；可重复调用。
func (s *Server) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		listener := s.listener
		conns := make([]*serverConn, 0, len(s.conns))
		for conn := range s.conns {
			conns = append(conns, conn)
		}
		s.conns = make(map[*serverConn]struct{})
		s.mu.Unlock()

		_ = listener.Close()
		for _, conn := range conns {
			conn.closeOnce.Do(func() { _ = conn.conn.Close() })
		}
		if s.ownDataDir {
			_ = os.RemoveAll(s.dataDir)
			return
		}
		_ = os.Remove(s.endpoint.Address)
	})
	return nil
}

func (s *Server) accept() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			_ = conn.Close()
			return
		}
		s.nextConnID++
		tracked := &serverConn{
			id:       s.nextConnID,
			conn:     conn,
			attached: make(map[string]bool),
		}
		s.conns[tracked] = struct{}{}
		s.accepted++
		s.mu.Unlock()
		go s.serveConn(tracked)
	}
}

func (s *Server) serveConn(conn *serverConn) {
	defer s.closeConn(conn)
	reader := bufio.NewReader(conn.conn)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			frame := bytes.TrimSuffix(line, []byte("\n"))
			frame = bytes.TrimSuffix(frame, []byte("\r"))
			if len(bytes.TrimSpace(frame)) > 0 {
				s.handleFrame(conn, frame)
			}
		}
		if err != nil {
			return
		}
	}
}

func (s *Server) closeConn(conn *serverConn) {
	conn.closeOnce.Do(func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
		_ = conn.conn.Close()
	})
}

func (s *Server) handleFrame(conn *serverConn, frame []byte) {
	now := time.Now()

	var incoming struct {
		ID     json.RawMessage `json:"id"`
		Method *string         `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	parsed := json.Unmarshal(frame, &incoming) == nil
	method := ""
	id := protocol.ID("")
	if parsed && incoming.Method != nil {
		method = *incoming.Method
	}
	if parsed && len(incoming.ID) > 0 {
		id = protocol.ID(bytes.TrimSpace(incoming.ID))
	}

	s.mu.Lock()
	received := Frame{
		At:        now,
		Conn:      conn.id,
		Direction: DirectionReceived,
		Method:    method,
		ID:        id,
		Raw:       append([]byte(nil), frame...),
	}
	s.frames = append(s.frames, received)
	occurrence := 0
	if parsed && method != "" {
		s.methodCounts[method]++
		occurrence = s.methodCounts[method]
		// 客户端通知没有 id，只算入帧记录，不算请求。
		if len(id) > 0 && string(id) != "null" {
			s.requests = append(s.requests, Request{
				At:         now,
				Conn:       conn.id,
				ID:         id,
				Method:     method,
				Params:     append(json.RawMessage(nil), incoming.Params...),
				Occurrence: occurrence,
			})
		}
	}
	fault := s.faultForLocked(method, occurrence)
	s.mu.Unlock()

	// 客户端通知没有 id，只记录不回执。
	if method == "" || len(id) == 0 || string(id) == "null" {
		return
	}
	go s.respond(conn, id, method, incoming.Params, fault)
}

// faultForLocked 取第一条命中的故障规则；调用方持有 Server.mu。
func (s *Server) faultForLocked(method string, occurrence int) Fault {
	for _, fault := range s.faults {
		if fault.Method != method {
			continue
		}
		if fault.Occurrence == 0 || fault.Occurrence == occurrence {
			return fault
		}
	}
	return Fault{}
}

func (s *Server) respond(
	conn *serverConn,
	id protocol.ID,
	method string,
	params json.RawMessage,
	fault Fault,
) {
	if fault.Delay > 0 {
		time.Sleep(fault.Delay)
	}
	if fault.DisconnectBefore {
		s.closeConn(conn)
		return
	}
	if fault.OversizeFrameBytes > 0 {
		if err := conn.write(s, oversizeFrame(fault.OversizeFrameBytes), "fake.oversize", ""); err != nil {
			return
		}
	}
	if fault.NoReply {
		return
	}

	var result any
	var rpcErr *protocol.RPCError
	if fault.UnknownMethod {
		rpcErr = unknownMethodError(method)
	} else {
		result, rpcErr = s.resolve(method, params, conn)
	}
	if rpcErr != nil {
		conn.writeError(s, id, method, rpcErr)
	} else {
		conn.writeResult(s, id, method, result)
	}
	if fault.DisconnectAfter {
		s.closeConn(conn)
	}
	if method == "daemon.shutdown" && s.script.StopOnShutdown {
		_ = s.Close()
	}
}

// resolve 给出一条请求的回执。脚本应答优先，其次才是内置处理。
func (s *Server) resolve(
	method string,
	params json.RawMessage,
	conn *serverConn,
) (any, *protocol.RPCError) {
	if reply, ok := s.replyByMethod[method]; ok {
		return reply.Result, reply.Error
	}
	switch method {
	case "hello":
		return s.helloResult()
	case "daemon.ping":
		return s.pingResult(), nil
	case "daemon.shutdown":
		return nil, nil
	case "session.attach":
		return s.attach(params, conn)
	case "session.detach":
		return s.detach(params, conn)
	case "session.history":
		return s.history(params)
	default:
		return nil, unknownMethodError(method)
	}
}

func (s *Server) helloResult() (any, *protocol.RPCError) {
	if s.script.HelloError != nil {
		return nil, s.script.HelloError
	}
	return protocol.HelloResult{
		InstanceID:      s.script.InstanceID,
		ServerVersion:   s.script.ServerVersion,
		ProtocolVersion: s.script.ProtocolVersion,
		DaemonProfile:   s.script.Profile,
		Features:        append([]string(nil), s.script.Features...),
	}, nil
}

func (s *Server) pingResult() protocol.PingResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return protocol.PingResult{
		Sessions:      len(s.sessions),
		Connections:   len(s.conns),
		ServerVersion: s.script.ServerVersion,
	}
}

func (s *Server) attach(
	params json.RawMessage,
	conn *serverConn,
) (any, *protocol.RPCError) {
	var attach protocol.SessionAttachParams
	if err := json.Unmarshal(params, &attach); err != nil {
		return nil, AppError(-32602, "invalid_params", "session.attach 参数无法解析: "+err.Error())
	}
	s.mu.Lock()
	state, ok := s.sessions[attach.SessionID]
	if !ok {
		s.mu.Unlock()
		return nil, AppError(-32000, "unknown_session", "未知会话 "+attach.SessionID)
	}
	conn.attached[attach.SessionID] = true
	head := state.head
	running := state.spec.Running
	branchID := state.spec.BranchID
	replay := append([]protocol.RawEvent(nil), state.spec.AttachReplay...)
	s.mu.Unlock()

	// 订阅重放先于 attach 回执，与真实服务端的顺序一致。
	for _, event := range replay {
		frame, err := eventFrame(attach.SessionID, branchID, event)
		if err != nil {
			continue
		}
		_ = conn.write(s, frame, "session.event", "")
	}
	return protocol.SessionAttachResult{
		SessionID: attach.SessionID,
		Sequence:  &head,
		Running:   &running,
	}, nil
}

func (s *Server) detach(
	params json.RawMessage,
	conn *serverConn,
) (any, *protocol.RPCError) {
	var detach protocol.SessionDetachParams
	if err := json.Unmarshal(params, &detach); err != nil {
		return nil, AppError(-32602, "invalid_params", "session.detach 参数无法解析: "+err.Error())
	}
	s.mu.Lock()
	delete(conn.attached, detach.SessionID)
	s.mu.Unlock()
	return nil, nil
}

func (s *Server) history(params json.RawMessage) (any, *protocol.RPCError) {
	var history protocol.SessionHistoryParams
	if err := json.Unmarshal(params, &history); err != nil {
		return nil, AppError(-32602, "invalid_params", "session.history 参数无法解析: "+err.Error())
	}

	s.mu.Lock()
	state, ok := s.sessions[history.SessionID]
	if !ok {
		s.mu.Unlock()
		return nil, AppError(-32000, "unknown_session", "未知会话 "+history.SessionID)
	}
	head := state.head
	latestTurnStart := state.spec.LatestTurnStart
	events := append([]protocol.RawEvent(nil), state.spec.Events...)
	s.mu.Unlock()

	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Sequence < events[j].Sequence
	})

	after := int64(0)
	if history.After != nil {
		after = *history.After
	}
	// latestTurn 的边界由样例元数据给出；事件词表未冻结前不猜轮次标记。
	if history.LatestTurn != nil && *history.LatestTurn && latestTurnStart > 0 && after < latestTurnStart {
		after = latestTurnStart - 1
	}
	through := int64(0)
	if history.Through != nil {
		through = *history.Through
	}
	limit := 200
	if history.Limit != nil && *history.Limit > 0 {
		limit = *history.Limit
	}
	if history.Before != nil && *history.Before > 0 {
		// 仅实现向前游标；向后翻页的语义等 D2 冻结后再补，见领域文档已知限制。
		return nil, AppError(-32602, "invalid_params", "假服务端暂不支持 before 游标")
	}

	selected := make([]protocol.RawEvent, 0, limit)
	remaining := 0
	for _, event := range events {
		if event.Sequence <= after {
			continue
		}
		if through > 0 && event.Sequence > through {
			continue
		}
		if len(selected) >= limit {
			remaining++
			continue
		}
		selected = append(selected, event)
	}
	nextAfter := after
	if len(selected) > 0 {
		nextAfter = selected[len(selected)-1].Sequence
	}
	return protocol.SessionHistoryResult{
		Events:    selected,
		Through:   head,
		NextAfter: nextAfter,
		Done:      remaining == 0,
	}, nil
}

func (c *serverConn) write(s *Server, frame []byte, method string, id protocol.ID) error {
	c.writeMu.Lock()
	_, err := c.conn.Write(frame)
	c.writeMu.Unlock()
	s.recordSent(c, method, id, frame)
	return err
}

func (c *serverConn) writeResult(s *Server, id protocol.ID, method string, result any) {
	frame, err := protocol.EncodeFrame(map[string]any{
		"jsonrpc": protocol.JSONRPCVersion,
		"id":      id,
		"result":  result,
	})
	if err != nil {
		// 脚本结果不可编码时回内部错误，避免静默无回执。
		c.writeError(s, id, method, AppError(-32603, "internal_error", "脚本回执无法编码: "+err.Error()))
		return
	}
	_ = c.write(s, frame, method, id)
}

func (c *serverConn) writeError(s *Server, id protocol.ID, method string, rpcErr *protocol.RPCError) {
	frame, err := protocol.EncodeFrame(map[string]any{
		"jsonrpc": protocol.JSONRPCVersion,
		"id":      id,
		"error":   rpcErr,
	})
	if err != nil {
		return
	}
	_ = c.write(s, frame, method, id)
}

func (s *Server) recordSent(conn *serverConn, method string, id protocol.ID, frame []byte) {
	s.mu.Lock()
	s.frames = append(s.frames, Frame{
		At:        time.Now(),
		Conn:      conn.id,
		Direction: DirectionSent,
		Method:    method,
		ID:        id,
		Raw:       append([]byte(nil), frame...),
	})
	s.mu.Unlock()
}

func notificationFrame(method string, params json.RawMessage) ([]byte, error) {
	frame, err := protocol.EncodeFrame(map[string]any{
		"jsonrpc": protocol.JSONRPCVersion,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return nil, fmt.Errorf("编码通知 %s: %w", method, err)
	}
	return frame, nil
}

func eventFrame(sessionID, branchID string, event protocol.RawEvent) ([]byte, error) {
	params, err := json.Marshal(protocol.SessionEventParams{
		SessionID: sessionID,
		BranchID:  branchID,
		Event:     event,
	})
	if err != nil {
		return nil, err
	}
	return notificationFrame("session.event", params)
}

// oversizeFrame 生成一条指定字节数以上的合法 JSON 通知帧。客户端按自己的
// MaxFrameBytes 决定丢弃；把上限调小即可在测试里稳定触发。
func oversizeFrame(size int) []byte {
	if size < 64 {
		size = 64
	}
	pad := make([]byte, size)
	for i := range pad {
		pad[i] = 'a'
	}
	frame, _ := json.Marshal(map[string]any{
		"jsonrpc": protocol.JSONRPCVersion,
		"method":  "fake.oversize",
		"params":  map[string]string{"pad": string(pad)},
	})
	return append(frame, '\n')
}

func cloneFrames(frames []Frame) []Frame {
	cloned := make([]Frame, 0, len(frames))
	for _, frame := range frames {
		cloned = append(cloned, cloneFrame(frame))
	}
	return cloned
}

func cloneFrame(frame Frame) Frame {
	frame.Raw = append([]byte(nil), frame.Raw...)
	return frame
}

func cloneRequests(requests []Request) []Request {
	cloned := make([]Request, 0, len(requests))
	for _, request := range requests {
		cloned = append(cloned, cloneRequest(request))
	}
	return cloned
}

func cloneRequest(request Request) Request {
	request.Params = append(json.RawMessage(nil), request.Params...)
	return request
}
