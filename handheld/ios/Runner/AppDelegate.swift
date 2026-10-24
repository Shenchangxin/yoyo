import Flutter
import UIKit
import UserNotifications

@main
@objc class AppDelegate: FlutterAppDelegate {
  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    GeneratedPluginRegistrant.register(with: self)
    let ok = super.application(application, didFinishLaunchingWithOptions: launchOptions)
    if let controller = window?.rootViewController as? FlutterViewController {
      let channel = FlutterMethodChannel(name: "yoyo/presence", binaryMessenger: controller.binaryMessenger)
      channel.setMethodCallHandler { call, result in
        switch call.method {
        case "start":
          let args = call.arguments as? [String: Any]
          let text = args?["text"] as? String ?? "工作站连接中"
          self.postPresence(text)
          result(nil)
        case "stop":
          UNUserNotificationCenter.current().removeDeliveredNotifications(withIdentifiers: ["yoyo.presence"])
          result(nil)
        default:
          result(FlutterMethodNotImplemented)
        }
      }
    }
    return ok
  }

  private func postPresence(_ text: String) {
    let center = UNUserNotificationCenter.current()
    center.requestAuthorization(options: [.alert, .sound]) { granted, _ in
      guard granted else { return }
      let content = UNMutableNotificationContent()
      content.title = "Yoyo"
      content.body = text
      let request = UNNotificationRequest(identifier: "yoyo.presence", content: content, trigger: nil)
      center.add(request)
    }
  }
}
