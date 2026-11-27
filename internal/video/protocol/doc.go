// Package protocol 把视频、图片、音频和文本生成供应商的接口翻译成统一的协议适配器。
//
// 适配器只翻译请求和响应：把生成请求编成请求规格，再把供应商的响应解析回来。这个包不发出网络请求，也不保存密钥；
// 真正发请求、拿密钥的是 internal/video 里的画布插件。
//
// 适配器有两种来源：随二进制带上的内置适配器（Builtins），以及从插件清单装出来的适配器（LoadInstalledProviders、
// ValidateManifest）。清单里的模板用少量 $ 运算符求值成供应商请求，插件 zip 由 ParsePluginPackage 检查，
// 两者都不执行插件里的代码。Registry 按标识登记适配器，并按能力和使用面列出元数据。
package protocol
