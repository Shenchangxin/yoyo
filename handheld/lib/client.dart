import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

import 'workstation.dart';

class YoyoClient extends ChangeNotifier {
  YoyoClient({Workstation? initial}) : _port = initial {
    if (initial != null) _listen(initial);
  }

  static const _presence = MethodChannel('yoyo/presence');

  Workstation? _port;
  var busy = false;
  String error = '';
  String notice = '';
  List<Map<String, dynamic>> sessions = [];
  List<String> running = [];
  StreamSubscription<Map<String, dynamic>>? _sub;

  Workstation? get port => _port;
  bool get linked => _port != null && _port!.online;
  String get where => _port?.label ?? '未连接';

  Future<void> connect(String raw, {String name = '手机'}) async {
    error = '';
    busy = true;
    notice = '正在连接，等待这台电脑允许';
    notifyListeners();
    try {
      final code = PairCode.parse(raw);
      if (!code.complete) {
        throw WorkstationException('配对码不完整。在电脑上重新出示。');
      }
      final remote = RemoteWorkstation();
      await remote.connect(code.uri, name: name);
      await _adopt(remote);
      notice = '';
      await refresh();
      await showPresence('已连接 ${remote.label}');
    } on WorkstationException catch (e) {
      error = e.message;
      notice = '';
    } catch (e) {
      error = '没有连上。$e';
      notice = '';
    } finally {
      busy = false;
      notifyListeners();
    }
  }

  Future<void> useLocal() async {
    error = '';
    notice = '';
    await _adopt(LocalWorkstation());
    await refresh();
  }

  Future<void> disconnect() async {
    final current = _port;
    await _sub?.cancel();
    _sub = null;
    if (current is RemoteWorkstation) await current.forget();
    await current?.close();
    _port = null;
    sessions = [];
    running = [];
    await hidePresence();
    notifyListeners();
  }

  Future<void> restore() async {
    if (busy || _port is LocalWorkstation) return;
    if (_port != null && _port!.online) return;
    final remote = RemoteWorkstation();
    if (!await remote.hasTarget()) {
      await remote.close();
      return;
    }
    error = '';
    busy = true;
    notice = '正在回到这台电脑';
    notifyListeners();
    try {
      final started = await remote.reconnect();
      if (!started) {
        notice = '';
        await remote.close();
        return;
      }
      await _adopt(remote);
      notice = '';
      await refresh();
      await showPresence('已连接 ${remote.label}');
    } on WorkstationException catch (e) {
      error = e.message;
      notice = '';
      await remote.close();
    } catch (e) {
      error = '没有连上。$e';
      notice = '';
      await remote.close();
    } finally {
      busy = false;
      notifyListeners();
    }
  }

  Future<dynamic> call(String method, Map<String, dynamic> params, {bool confirm = false}) async {
    final current = _port;
    if (current == null) throw WorkstationException('先连接一台电脑，或在这台手机上预备。');
    try {
      final out = await current.call(method, params, confirm: confirm);
      error = '';
      return out;
    } on WorkstationException catch (e) {
      error = e.message;
      notifyListeners();
      rethrow;
    }
  }

  Future<void> refresh() async {
    if (_port == null) return;
    try {
      final raw = await _port!.call('thread.list', {});
      sessions = _asMaps(raw);
      error = '';
    } on WorkstationException catch (e) {
      error = e.message;
    }
    try {
      final ids = await _port!.call('turn.running_ids', {});
      if (ids is List) running = ids.map((id) => '$id').toList();
    } catch (_) {}
    notifyListeners();
  }

  Future<void> showPresence(String text) => _presenceStart(text);

  Future<void> hidePresence() => _presenceStop();

  Future<void> _adopt(Workstation next) async {
    await _port?.close();
    await _sub?.cancel();
    _port = next;
    _listen(next);
    notifyListeners();
  }

  void _listen(Workstation next) {
    _sub = next.events.listen((event) {
      if (event['topic'] == 'link') {
        unawaited(restore());
        return;
      }
      if (event['type'] == 'live') return;
      unawaited(refresh());
    });
  }

  Future<void> _presenceStart(String text) async {
    try {
      await _presence.invokeMethod('start', {'text': text});
    } catch (_) {}
  }

  Future<void> _presenceStop() async {
    try {
      await _presence.invokeMethod('stop');
    } catch (_) {}
  }

  @override
  void dispose() {
    unawaited(_port?.close() ?? Future.value());
    unawaited(_sub?.cancel() ?? Future.value());
    super.dispose();
  }
}

List<Map<String, dynamic>> _asMaps(dynamic raw) {
  if (raw is List) {
    return raw.whereType<Map>().map((m) => m.map((k, v) => MapEntry('$k', v))).toList();
  }
  if (raw is Map && raw['sessions'] is List) {
    return _asMaps(raw['sessions']);
  }
  return [];
}
