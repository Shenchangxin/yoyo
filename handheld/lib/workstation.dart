import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:cryptography/cryptography.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

const signDomain = 'yoyo.remote.v1';

class WorkstationException implements Exception {
  WorkstationException(this.message);
  final String message;
  @override
  String toString() => message;
}

abstract class Workstation {
  String get label;
  bool get online;
  Future<dynamic> call(String method, Map<String, dynamic> params, {bool confirm = false});
  Future<Uint8List> media(String path);
  Stream<Map<String, dynamic>> get events;
  Future<void> close();
}

class LocalWorkstation implements Workstation {
  final List<Map<String, dynamic>> _sessions = [];
  final _events = StreamController<Map<String, dynamic>>.broadcast();
  var _seq = 0;

  Map<String, dynamic>? _find(String id) => _findIn(_sessions, id);

  @override
  String get label => '这台手机';

  @override
  bool get online => true;

  @override
  Stream<Map<String, dynamic>> get events => _events.stream;

  @override
  Future<void> close() async {}

  @override
  Future<Uint8List> media(String path) {
    throw WorkstationException('文件在电脑上。这台手机的循环还没接上。');
  }

  @override
  Future<dynamic> call(String method, Map<String, dynamic> params, {bool confirm = false}) async {
    switch (method) {
      case 'thread.list':
      case 'session.list':
        return _sessions.map((s) => Map<String, dynamic>.from(s)).toList();
      case 'thread.start':
        final id = 'local-${++_seq}';
        final session = {
          'id': id,
          'title': '新的对话',
          'preview': '',
          'model': '未接入',
          'channel': params['channel'] ?? 'agent',
        };
        _sessions.insert(0, session);
        _events.add({'type': 'state', 'topic': 'session', 'object': id});
        return session;
      case 'thread.model.set':
        final row = _find('${params['session']}');
        if (row == null) throw WorkstationException('没有这条对话');
        row['model'] = params['model'];
        _events.add({'type': 'state', 'topic': 'session', 'object': row['id']});
        return row;
      case 'thread.rename':
        final row = _find('${params['session']}');
        if (row == null) throw WorkstationException('没有这条对话');
        row['title'] = params['title'];
        return row;
      case 'turn.start':
        return {
          'ok': true,
          'local': true,
          'text': '独立循环还没接上。这句话只留在这台手机上，没有送到电脑。',
        };
      case 'health':
        return {'ok': true, 'local': true};
      default:
        return {
          'ok': true,
          'local': true,
          'method': method,
          'note': '独立循环还没接上。这个操作没有送到电脑。',
        };
    }
  }
}

class RemoteWorkstation implements Workstation {
  RemoteWorkstation({FlutterSecureStorage? storage}) : _storage = storage ?? const FlutterSecureStorage();

  final FlutterSecureStorage _storage;
  final _events = StreamController<Map<String, dynamic>>.broadcast();
  final _pending = <String, Completer<Map<String, dynamic>>>{};
  WebSocket? _socket;
  SimpleKeyPair? _keys;
  var _online = false;
  var _name = '电脑';
  var _host = '';
  var _portNum = 0;
  var _fp = '';
  var _deviceID = '';

  @override
  String get label => _name;

  @override
  bool get online => _online;

  @override
  Stream<Map<String, dynamic>> get events => _events.stream;

  static Future<SimpleKeyPair> loadOrCreate(FlutterSecureStorage storage) async {
    final algorithm = Ed25519();
    final existing = await storage.read(key: 'yoyo.device.seed');
    if (existing != null && existing.isNotEmpty) {
      return algorithm.newKeyPairFromSeed(base64Decode(existing));
    }
    final created = await algorithm.newKeyPair();
    final data = await created.extract();
    await storage.write(key: 'yoyo.device.seed', value: base64Encode(data.bytes));
    return created;
  }

  Future<void> connect(Uri code, {required String name}) async {
    final host = code.queryParameters['h'] ?? '';
    final port = int.tryParse(code.queryParameters['p'] ?? '') ?? 0;
    final fp = (code.queryParameters['fp'] ?? '').toLowerCase();
    final nonce = code.queryParameters['n'] ?? '';
    await _open(host, port, fp, name: name, resume: false, pairNonce: nonce);
    await _remember(host, port, fp);
  }

  Future<bool> hasTarget() async {
    final host = await _storage.read(key: 'yoyo.remote.host') ?? '';
    final port = int.tryParse(await _storage.read(key: 'yoyo.remote.port') ?? '') ?? 0;
    final fp = await _storage.read(key: 'yoyo.remote.fp') ?? '';
    return host.isNotEmpty && port != 0 && fp.isNotEmpty;
  }

  Future<bool> reconnect({String name = '手机'}) async {
    final host = await _storage.read(key: 'yoyo.remote.host') ?? '';
    final port = int.tryParse(await _storage.read(key: 'yoyo.remote.port') ?? '') ?? 0;
    final fp = (await _storage.read(key: 'yoyo.remote.fp') ?? '').toLowerCase();
    if (host.isEmpty || port == 0 || fp.isEmpty) return false;
    await _open(host, port, fp, name: name, resume: true, pairNonce: '');
    return true;
  }

  Future<void> forget() async {
    await _storage.delete(key: 'yoyo.remote.host');
    await _storage.delete(key: 'yoyo.remote.port');
    await _storage.delete(key: 'yoyo.remote.fp');
  }

  Future<void> _remember(String host, int port, String fp) async {
    await _storage.write(key: 'yoyo.remote.host', value: host);
    await _storage.write(key: 'yoyo.remote.port', value: '$port');
    await _storage.write(key: 'yoyo.remote.fp', value: fp);
  }

  Future<void> _open(String host, int port, String fp, {required String name, required bool resume, required String pairNonce}) async {
    if (host.isEmpty || port == 0 || fp.isEmpty || (!resume && pairNonce.isEmpty)) {
      throw WorkstationException('配对码不完整。在电脑上重新出示。');
    }
    _keys = await RemoteWorkstation.loadOrCreate(_storage);
    final client = HttpClient()
      ..connectionTimeout = const Duration(seconds: 8)
      ..badCertificateCallback = (cert, _, __) {
        final digest = sha256.convert(cert.der).toString();
        return digest == fp;
      };
    try {
      _socket = await WebSocket.connect('wss://$host:$port/remote/ws', customClient: client);
    } on SocketException {
      throw WorkstationException('连不上这台电脑。确认手机和电脑在同一局域网；访客网络请让电脑加入手机热点。');
    } on HandshakeException {
      throw WorkstationException('证书指纹和配对码不一致。在电脑上重新出示配对码。');
    } catch (e) {
      throw WorkstationException('连不上这台电脑。$e');
    }
    _socket!.listen(_onData, onError: (Object e) {
      _online = false;
      _events.add({'type': 'state', 'topic': 'link'});
    }, onDone: () {
      _online = false;
      _events.add({'type': 'state', 'topic': 'link'});
    });
    final pending = _next((m) => m['type'] == 'pair_ok' || m['type'] == 'error');
    final pub = await _keys!.extractPublicKey();
    final nonce = resume ? _nonce() : pairNonce;
    final kind = resume ? 'resume' : 'pair';
    final message = utf8.encode('$signDomain\n$kind\n$nonce');
    final signature = await Ed25519().sign(message, keyPair: _keys!);
    _socket!.add(jsonEncode({
      'type': kind,
      'name': name,
      'n': nonce,
      'pub': base64Encode(pub.bytes),
      'sig': base64Encode(signature.bytes),
    }));
    final frame = await pending;
    if (frame['type'] == 'error') {
      await _socket?.close();
      _socket = null;
      final messageText = '${frame['error']}';
      if (messageText.contains('还没被允许')) await forget();
      throw WorkstationException(messageText);
    }
    _online = true;
    _name = host;
    _host = host;
    _portNum = port;
    _fp = fp;
    _deviceID = '${frame['id'] ?? ''}';
  }

  @override
  Future<Uint8List> media(String path) async {
    final keys = _keys;
    if (!_online || keys == null || _deviceID.isEmpty) {
      throw WorkstationException('工作站不在线上。回到配对，或改在这台手机上预备。');
    }
    final ts = DateTime.now().millisecondsSinceEpoch ~/ 1000;
    final nonce = _nonce();
    final message = utf8.encode('$signDomain\nget\n/remote/media\n$path\n$ts\n$nonce');
    final signature = await Ed25519().sign(message, keyPair: keys);
    final client = HttpClient()
      ..connectionTimeout = const Duration(seconds: 8)
      ..badCertificateCallback = (cert, _, __) => sha256.convert(cert.der).toString() == _fp;
    final uri = Uri(scheme: 'https', host: _host, port: _portNum, path: '/remote/media', queryParameters: {'path': path});
    try {
      final request = await client.getUrl(uri);
      request.headers.set('X-Yoyo-Device', _deviceID);
      request.headers.set('X-Yoyo-Ts', '$ts');
      request.headers.set('X-Yoyo-Nonce', nonce);
      request.headers.set('X-Yoyo-Sig', base64Encode(signature.bytes));
      final response = await request.close();
      final bytes = await response.fold<List<int>>([], (acc, chunk) => acc..addAll(chunk));
      if (response.statusCode != 200) {
        throw WorkstationException(utf8.decode(bytes, allowMalformed: true));
      }
      return Uint8List.fromList(bytes);
    } on WorkstationException {
      rethrow;
    } on SocketException {
      throw WorkstationException('连不上这台电脑。确认手机和电脑在同一局域网；访客网络请让电脑加入手机热点。');
    } finally {
      client.close();
    }
  }

  Future<Map<String, dynamic>> _next(bool Function(Map<String, dynamic>) match) {
    final waiter = Completer<Map<String, dynamic>>();
    late StreamSubscription<Map<String, dynamic>> sub;
    sub = _events.stream.listen((event) {
      if (!waiter.isCompleted && match(event)) {
        waiter.complete(event);
        sub.cancel();
      }
    });
    return waiter.future.timeout(const Duration(minutes: 2), onTimeout: () {
      sub.cancel();
      throw WorkstationException('配对码过期了。在电脑上重新出示。');
    });
  }

  void _onData(dynamic raw) {
    final decoded = jsonDecode(raw as String);
    if (decoded is! Map) return;
    final frame = decoded.map((k, v) => MapEntry('$k', v));
    if (frame['type'] == 'rpc') {
      final id = '${frame['id']}';
      _pending.remove(id)?.complete(frame);
      return;
    }
    _events.add(frame);
  }

  @override
  Future<dynamic> call(String method, Map<String, dynamic> params, {bool confirm = false}) async {
    final socket = _socket;
    final keys = _keys;
    if (socket == null || keys == null || !_online) {
      throw WorkstationException('工作站不在线上。回到配对，或改在这台手机上预备。');
    }
    final id = '${DateTime.now().microsecondsSinceEpoch}';
    final paramsText = jsonEncode(params);
    final ts = DateTime.now().millisecondsSinceEpoch ~/ 1000;
    final nonce = _nonce();
    final sum = sha256.convert(utf8.encode(paramsText)).toString();
    final message = utf8.encode('$signDomain\nrpc\n$id\n$method\n$sum\n$ts\n$nonce');
    final signature = await Ed25519().sign(message, keyPair: keys);
    final done = Completer<Map<String, dynamic>>();
    _pending[id] = done;
    final head = jsonEncode({
      'type': 'rpc',
      'id': id,
      'method': method,
      'ts': ts,
      'nonce': nonce,
      'sig': base64Encode(signature.bytes),
      'confirm': confirm,
    });
    socket.add('${head.substring(0, head.length - 1)},"params":$paramsText}');
    final frame = await done.future.timeout(const Duration(seconds: 30), onTimeout: () {
      _pending.remove(id);
      throw WorkstationException('电脑没有在 30 秒内回答。');
    });
    if (frame['error'] != null) throw WorkstationException('${frame['error']}');
    return frame['result'];
  }

  @override
  Future<void> close() async {
    _online = false;
    await _socket?.close();
    _socket = null;
  }

  String _nonce() {
    final bytes = List<int>.generate(12, (_) => Random.secure().nextInt(256));
    return base64UrlEncode(bytes);
  }
}

Map<String, dynamic>? _findIn(List<Map<String, dynamic>> sessions, String id) {
  for (final session in sessions) {
    if (session['id'] == id) return session;
  }
  return null;
}

class PairCode {
  PairCode.parse(String raw) : uri = Uri.parse(raw.trim());
  final Uri uri;
  String get fingerprint => uri.queryParameters['fp'] ?? '';
  bool get complete => uri.queryParameters['h'] != null && uri.queryParameters['n'] != null && fingerprint.isNotEmpty;
}
