import CoreGraphics
import Foundation

guard CommandLine.arguments.count > 1, let pid = Int32(CommandLine.arguments[1]) else {
    FileHandle.standardError.write("usage: windowid <pid>\n".data(using: .utf8)!)
    exit(2)
}
let options: CGWindowListOption = [.optionOnScreenOnly, .excludeDesktopElements]
guard let list = CGWindowListCopyWindowInfo(options, kCGNullWindowID) as? [[String: Any]] else {
    exit(1)
}
for window in list {
    guard let owner = window[kCGWindowOwnerPID as String] as? Int32, owner == pid else { continue }
    guard let layer = window[kCGWindowLayer as String] as? Int, layer == 0 else { continue }
    guard let number = window[kCGWindowNumber as String] as? Int else { continue }
    guard let bounds = window[kCGWindowBounds as String] as? [String: Any] else { continue }
    let width = bounds["Width"] as? Double ?? 0
    let height = bounds["Height"] as? Double ?? 0
    guard width > 200, height > 200 else { continue }
    print("\(number)\t\(Int(width))x\(Int(height))")
}
