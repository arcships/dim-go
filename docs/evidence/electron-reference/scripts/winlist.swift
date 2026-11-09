import CoreGraphics
import Foundation

guard CommandLine.arguments.count > 1, let want = Int(CommandLine.arguments[1]) else {
    exit(2)
}
let opts: CGWindowListOption = [.optionAll, .excludeDesktopElements]
guard let list = CGWindowListCopyWindowInfo(opts, kCGNullWindowID) as? [[String: Any]] else {
    exit(1)
}
for w in list {
    guard let pid = w[kCGWindowOwnerPID as String] as? Int, pid == want else { continue }
    let num = w[kCGWindowNumber as String] as? Int ?? -1
    let layer = w[kCGWindowLayer as String] as? Int ?? -1
    let name = w[kCGWindowName as String] as? String ?? ""
    let owner = w[kCGWindowOwnerName as String] as? String ?? ""
    var bounds = "0,0,0,0"
    if let b = w[kCGWindowBounds as String] as? [String: Any],
       let x = b["X"] as? Double, let y = b["Y"] as? Double,
       let wd = b["Width"] as? Double, let ht = b["Height"] as? Double {
        bounds = "\(Int(x)),\(Int(y)),\(Int(wd)),\(Int(ht))"
    }
    print("\(num)\t\(layer)\t\(bounds)\t\(owner)\t\(name)")
}
