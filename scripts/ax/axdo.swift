import AppKit
import ApplicationServices

func attr(_ el: AXUIElement, _ name: String) -> AnyObject? {
    var value: CFTypeRef?
    let err = AXUIElementCopyAttributeValue(el, name as CFString, &value)
    return err == .success ? value : nil
}

func children(_ el: AXUIElement) -> [AXUIElement] {
    (attr(el, kAXChildrenAttribute) as? [AXUIElement]) ?? []
}

func matches(_ el: AXUIElement, _ needle: String) -> Bool {
    let fields = [
        attr(el, kAXTitleAttribute) as? String ?? "",
        attr(el, kAXDescriptionAttribute) as? String ?? "",
        attr(el, kAXValueAttribute) as? String ?? "",
    ]
    return fields.contains { $0 == needle }
}

func findAll(_ el: AXUIElement, _ needle: String, into out: inout [AXUIElement], depth: Int = 0) {
    if depth > 12 { return }
    if matches(el, needle) { out.append(el) }
    for child in children(el) { findAll(child, needle, into: &out, depth: depth + 1) }
}

func press(_ el: AXUIElement) -> AXError {
    AXUIElementPerformAction(el, kAXPressAction as CFString)
}

func main() {
    let args = CommandLine.arguments
    guard args.count >= 3, let pid = pid_t(args[2]) else {
        print("usage: axdo <click|dump|set|focus|read> <pid> [text]")
        exit(2)
    }
    let app = AXUIElementCreateApplication(pid)
    switch args[1] {
    case "dump":
        printTree(app, depth: 0)
    case "click":
        guard args.count >= 4 else { print("click needs a label"); exit(2) }
        var found: [AXUIElement] = []
        findAll(app, args[3], into: &found)
        guard let target = found.first else {
            print("not found: \(args[3])")
            exit(1)
        }
        let err = press(target)
        print("press \(args[3]) -> \(err.rawValue)")
        exit(err == .success ? 0 : 1)
    case "set":
        guard args.count >= 5 else { print("set needs a label and text"); exit(2) }
        var found: [AXUIElement] = []
        findAll(app, args[3], into: &found)
        guard let target = found.first else {
            print("not found: \(args[3])")
            exit(1)
        }
        let err = AXUIElementSetAttributeValue(target, kAXValueAttribute as CFString, args[4] as CFString)
        print("set \(args[3]) -> \(err.rawValue)")
        exit(err == .success ? 0 : 1)
    case "rows":
        guard args.count >= 4 else { print("rows needs a table desc"); exit(2) }
        var tables: [AXUIElement] = []
        findAll(app, args[3], into: &tables)
        guard let table = tables.first(where: { (attr($0, kAXRoleAttribute) as? String) == "AXTable" }) else {
            print("table not found: \(args[3])"); exit(1)
        }
        var index = 0
        for row in children(table) {
            let desc = attr(row, kAXDescriptionAttribute) as? String ?? ""
            if (attr(row, kAXRoleAttribute) as? String) == "AXRow" {
                let selected = (attr(row, kAXSelectedAttribute) as? Bool) ?? false
                print("\(index)\t\(desc)\tselected=\(selected)")
                index += 1
            }
        }
    case "clickrow":
        guard args.count >= 5, let want = Int(args[4]) else { print("clickrow needs a table desc and index"); exit(2) }
        var tables: [AXUIElement] = []
        findAll(app, args[3], into: &tables)
        guard let table = tables.first(where: { (attr($0, kAXRoleAttribute) as? String) == "AXTable" }) else {
            print("table not found: \(args[3])"); exit(1)
        }
        var rows: [AXUIElement] = []
        for child in children(table) where (attr(child, kAXRoleAttribute) as? String) == "AXRow" {
            rows.append(child)
        }
        guard want >= 0 && want < rows.count else { print("row \(want) of \(rows.count)"); exit(1) }
        let err = press(rows[want])
        print("row \(want) -> \(err.rawValue)")
        exit(err == .success ? 0 : 1)
    case "frame":
        guard args.count >= 4 else { print("frame needs a label"); exit(2) }
        var found: [AXUIElement] = []
        findAll(app, args[3], into: &found)
        guard let target = found.first else { print("not found: \(args[3])"); exit(1) }
        var posRef: CFTypeRef?
        var sizeRef: CFTypeRef?
        AXUIElementCopyAttributeValue(target, kAXPositionAttribute as CFString, &posRef)
        AXUIElementCopyAttributeValue(target, kAXSizeAttribute as CFString, &sizeRef)
        var pos = CGPoint.zero
        var size = CGSize.zero
        if let posRef { AXValueGetValue(posRef as! AXValue, .cgPoint, &pos) }
        if let sizeRef { AXValueGetValue(sizeRef as! AXValue, .cgSize, &size) }
        print("\(pos.x) \(pos.y) \(size.width) \(size.height)")
    case "clickat":
        guard args.count >= 5, let x = Double(args[3]), let y = Double(args[4]) else {
            print("clickat needs x y"); exit(2)
        }
        let point = CGPoint(x: x, y: y)
        let down = CGEvent(mouseEventSource: nil, mouseType: .leftMouseDown, mouseCursorPosition: point, mouseButton: .left)
        let up = CGEvent(mouseEventSource: nil, mouseType: .leftMouseUp, mouseCursorPosition: point, mouseButton: .left)
        down?.post(tap: .cghidEventTap)
        usleep(80_000)
        up?.post(tap: .cghidEventTap)
        print("clicked \(x),\(y)")
    case "rightclickat":
        guard args.count >= 5, let x = Double(args[3]), let y = Double(args[4]) else {
            print("rightclickat needs x y"); exit(2)
        }
        let point = CGPoint(x: x, y: y)
        CGEvent(mouseEventSource: nil, mouseType: .rightMouseDown, mouseCursorPosition: point, mouseButton: .right)?.post(tap: .cghidEventTap)
        usleep(80_000)
        CGEvent(mouseEventSource: nil, mouseType: .rightMouseUp, mouseCursorPosition: point, mouseButton: .right)?.post(tap: .cghidEventTap)
        print("rightclicked \(x),\(y)")
    case "focused":
        if let f = attr(app, kAXFocusedUIElementAttribute) {
            let el = f as! AXUIElement
            let role = attr(el, kAXRoleAttribute) as? String ?? ""
            let title = attr(el, kAXTitleAttribute) as? String ?? ""
            let desc = attr(el, kAXDescriptionAttribute) as? String ?? ""
            print("\(role) | \(title) | \(desc)")
        } else { print("no focused element") }
    case "move":
        guard args.count >= 5, let x = Double(args[3]), let y = Double(args[4]) else {
            print("move needs x y")
            exit(2)
        }
        CGEvent(mouseEventSource: nil, mouseType: .mouseMoved,
                mouseCursorPosition: CGPoint(x: x, y: y), mouseButton: .left)?
            .post(tap: .cghidEventTap)
        print("moved \(x),\(y)")
    case "key":
        guard args.count >= 4, let code = UInt16(args[3]) else { print("key needs a keycode"); exit(2) }
        let down = CGEvent(keyboardEventSource: nil, virtualKey: code, keyDown: true)
        let up = CGEvent(keyboardEventSource: nil, virtualKey: code, keyDown: false)
        down?.post(tap: .cghidEventTap)
        usleep(60_000)
        up?.post(tap: .cghidEventTap)
        print("key \(code)")
    case "selectrow":
        guard args.count >= 5, let want = Int(args[4]) else { print("selectrow needs a table desc and index"); exit(2) }
        var tables: [AXUIElement] = []
        findAll(app, args[3], into: &tables)
        guard let table = tables.first(where: { (attr($0, kAXRoleAttribute) as? String) == "AXTable" }) else {
            print("table not found: \(args[3])"); exit(1)
        }
        var rows: [AXUIElement] = []
        for child in children(table) where (attr(child, kAXRoleAttribute) as? String) == "AXRow" {
            rows.append(child)
        }
        guard want >= 0 && want < rows.count else { print("row \(want) of \(rows.count)"); exit(1) }
        let err = AXUIElementSetAttributeValue(rows[want], kAXSelectedAttribute as CFString, kCFBooleanTrue)
        print("select row \(want) -> \(err.rawValue)")
        exit(err == .success ? 0 : 1)
    case "read":
        guard args.count >= 4 else { print("read needs a label"); exit(2) }
        var found: [AXUIElement] = []
        findAll(app, args[3], into: &found)
        guard let target = found.first else {
            print("not found: \(args[3])")
            exit(1)
        }
        print((attr(target, kAXValueAttribute) as? String) ?? "")
    default:
        print("unknown command")
        exit(2)
    }
}

func printTree(_ el: AXUIElement, depth: Int) {
    if depth > 10 { return }
    let role = attr(el, kAXRoleAttribute) as? String ?? "?"
    let title = attr(el, kAXTitleAttribute) as? String ?? ""
    let desc = attr(el, kAXDescriptionAttribute) as? String ?? ""
    let value = attr(el, kAXValueAttribute) as? String ?? ""
    let pad = String(repeating: "  ", count: depth)
    print("\(pad)\(role) title=\(title.debugDescription) desc=\(desc.debugDescription) value=\(value.prefix(60).debugDescription)")
    for child in children(el) { printTree(child, depth: depth + 1) }
}

main()
