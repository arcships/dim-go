// walkthrough-probe：走查脚本的桌面操作工具（只做取证，不改产品）。
//
// 子命令：
//
//	list <pid>              列出该进程的窗口：编号、位置、尺寸（像素与点）
//	click <x> <y>           在屏幕坐标（点）点一下
//	move <x> <y>            把指针移到屏幕坐标（点）
//	type <文本>             发送一段文本按键（ASCII 可见字符）
//	keys <键码>[,<键码>…]    按 macOS 虚拟键码发送按键（支持带修饰的组合，
//	                        例如 56,49 是 Shift+Space）
//	ocr <png>              识别图上的文字，逐行输出「x y 宽 高 置信度 文本」
//	                        （坐标是图片像素，原点左上）
//
// 坐标一律用「点」（DIP），与界面截图里量出来的逻辑像素一致。
import Foundation
import CoreGraphics
import ImageIO
import Vision

func fail(_ message: String) -> Never {
	FileHandle.standardError.write((message + "\n").data(using: .utf8)!)
	exit(1)
}

func windows(of pid: pid_t) -> [[String: Any]] {
	let options: CGWindowListOption = [.optionAll]
	guard let list = CGWindowListCopyWindowInfo(options, kCGNullWindowID) as? [[String: Any]] else { return [] }
	return list.filter { ($0[kCGWindowOwnerPID as String] as? pid_t) == pid }
}

func rect(_ info: [String: Any], _ key: String) -> CGRect {
	guard let d = info[key] as? [String: Any] else { return .zero }
	return CGRect(x: d["X"] as? CGFloat ?? 0, y: d["Y"] as? CGFloat ?? 0,
	              width: d["Width"] as? CGFloat ?? 0, height: d["Height"] as? CGFloat ?? 0)
}

func list(_ pid: pid_t) {
	for w in windows(of: pid) {
		let num = w[kCGWindowNumber as String] as? Int ?? 0
		let owner = w[kCGWindowOwnerName as String] as? String ?? ""
		let name = w[kCGWindowName as String] as? String ?? ""
		let layer = w[kCGWindowLayer as String] as? Int ?? 0
		if layer != 0 { continue }
		let b = rect(w, kCGWindowBounds as String)
		print("\(num)\t\(owner)\t\(name)\t\(Int(b.origin.x))\t\(Int(b.origin.y))\t\(Int(b.size.width))\t\(Int(b.size.height))")
	}
}

// ocr 识别一张图上的文字：坐标从 Vision 的归一化左下原点换成图片像素左上原点。
func ocr(_ path: String) {
	let url = URL(fileURLWithPath: path) as CFURL
	guard let src = CGImageSourceCreateWithURL(url, nil),
		let image = CGImageSourceCreateImageAtIndex(src, 0, nil) else {
		fail("读不了图片 \(path)")
	}
	let width = CGFloat(image.width), height = CGFloat(image.height)
	let request = VNRecognizeTextRequest()
	request.recognitionLevel = .accurate
	request.recognitionLanguages = ["zh-Hans", "en-US"]
	request.usesLanguageCorrection = true
	guard let handler = try? VNImageRequestHandler(cgImage: image, options: [:]) else {
		fail("视觉框架初始化失败")
	}
	do { try handler.perform([request]) } catch { fail("识别失败：\(error)") }
	for case let observation as VNRecognizedTextObservation in request.results ?? [] {
		guard let best = observation.topCandidates(1).first else { continue }
		let box = observation.boundingBox
		let x = box.minX * width
		let y = (1 - box.maxY) * height
		let w = box.width * width
		let h = box.height * height
		print(String(format: "%.0f\t%.0f\t%.0f\t%.0f\t%.2f\t%@", x, y, w, h, best.confidence, best.string))
	}
}

func post(_ type: CGEventType, _ point: CGPoint, _ button: CGMouseButton = .left) {
	let e = CGEvent(mouseEventSource: nil, mouseType: type, mouseCursorPosition: point, mouseButton: button)
	e?.post(tap: .cghidEventTap)
	usleep(30_000)
}

func click(_ x: Double, _ y: Double) {
	let p = CGPoint(x: x, y: y)
	post(.mouseMoved, p)
	post(.leftMouseDown, p)
	post(.leftMouseUp, p)
}

func type(_ text: String) {
	// 用 unicode 字符串直接投递，不依赖键位表：中文与符号也能过。
	for ch in text.unicodeScalars {
		let src = CGEventSource(stateID: .hidSystemState)
		let down = CGEvent(keyboardEventSource: src, virtualKey: 0, keyDown: true)
		let up = CGEvent(keyboardEventSource: src, virtualKey: 0, keyDown: false)
		var u = UniChar(ch.value)
		down?.keyboardSetUnicodeString(stringLength: 1, unicodeString: &u)
		up?.keyboardSetUnicodeString(stringLength: 1, unicodeString: &u)
		down?.post(tap: .cghidEventTap)
		up?.post(tap: .cghidEventTap)
		usleep(25_000)
	}
}

func keys(_ codes: [CGKeyCode]) {
	let src = CGEventSource(stateID: .hidSystemState)
	for code in codes {
		CGEvent(keyboardEventSource: src, virtualKey: code, keyDown: true)?.post(tap: .cghidEventTap)
		usleep(20_000)
		CGEvent(keyboardEventSource: src, virtualKey: code, keyDown: false)?.post(tap: .cghidEventTap)
		usleep(20_000)
	}
}

let args = CommandLine.arguments
guard args.count >= 2 else { fail("用法见文件头注释") }
switch args[1] {
case "list":
	guard args.count >= 3, let pid = pid_t(args[2]) else { fail("list 需要 pid") }
	list(pid)
case "click":
	guard args.count >= 4, let x = Double(args[2]), let y = Double(args[3]) else { fail("click 需要 x y") }
	click(x, y)
case "move":
	guard args.count >= 4, let x = Double(args[2]), let y = Double(args[3]) else { fail("move 需要 x y") }
	post(.mouseMoved, CGPoint(x: x, y: y))
case "type":
	guard args.count >= 3 else { fail("type 需要文本") }
	type(args[2])
case "keys":
	guard args.count >= 3 else { fail("keys 需要键码") }
	let codes = args[2].split(separator: ",").compactMap { CGKeyCode($0) }
	keys(codes)
case "ocr":
	guard args.count >= 3 else { fail("ocr 需要图片路径") }
	ocr(args[2])
default:
	fail("未知子命令 \(args[1])")
}
