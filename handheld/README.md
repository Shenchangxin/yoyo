# Yoyo 手机

这是电脑上那条循环的手机门。活在电脑上干。手机和电脑要在同一个局域网：同一张 Wi-Fi，或让电脑加入手机热点。

完整说明在仓库的 [README.zh-CN.md](../README.zh-CN.md#手机) 和 [README.md](../README.md#phone)。

## 安装 Android

1. 从 [Releases](https://github.com/Shenchangxin/yoyo/releases/latest) 下载 `Yoyo-*-android.apk`。CI 的 Actions 产物里也有。
2. 传到手机，允许安装未知应用，打开它。
3. 电脑上打开 Yoyo，进入 **设置 → 高级 → 手机**，点 **显示配对码**。
4. 用这个 App 扫码。回到电脑点 **允许**。
5. 新建对话、切换模型和其他操作都会在电脑上生效。

配对码两分钟有效，只能用一次。签名换了就先卸载再装。

iOS 要在 Mac 上用 Xcode 和 Apple 签名构建，CI 不提供可安装的 IPA。鸿蒙服务卡片在 [`harmony/`](harmony/)，需要 DevEco Studio 和 flutter-oh，CI 不发布 HAP。

## 不装 App

用微信或其他扫码工具扫电脑上的同一个码。浏览器会停在等待页。电脑上允许名为「浏览器」的那一项之后，手机浏览器就是电脑上的那套界面，操作写回电脑。这次允许大约 12 小时。只接受内网地址，存着的密钥不会发到手机上。

## 本地开发

Flutter 3.29.x，Dart SDK `^3.7.2`。

```bash
flutter pub get
flutter analyze
flutter test
flutter build apk --release
```

安装包在 `build/app/outputs/flutter-apk/app-release.apk`。没有 `android/keystore.properties` 时，release 用 debug 签名，方便侧载。
