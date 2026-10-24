import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';

import 'client.dart';
import 'methods.g.dart';
import 'scan.dart';
import 'surfaces.dart';
import 'theme.dart';
import 'workstation.dart';

class YoyoApp extends StatefulWidget {
  const YoyoApp({super.key, this.client});
  final YoyoClient? client;

  @override
  State<YoyoApp> createState() => _YoyoAppState();
}

class _YoyoAppState extends State<YoyoApp> with WidgetsBindingObserver {
  late final YoyoClient client = widget.client ?? YoyoClient();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    if (widget.client == null) unawaited(client.restore());
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed && widget.client == null) {
      unawaited(client.restore());
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    if (widget.client == null) client.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Yoyo',
      debugShowCheckedModeBanner: false,
      theme: yoyoTheme(),
      home: AnimatedBuilder(
        animation: client,
        builder: (context, _) {
          if (client.port == null) return PairScreen(client: client);
          return WorkstationShell(client: client);
        },
      ),
    );
  }
}

class PairScreen extends StatefulWidget {
  const PairScreen({super.key, required this.client});
  final YoyoClient client;

  @override
  State<PairScreen> createState() => _PairScreenState();
}

class _PairScreenState extends State<PairScreen> {
  final _code = TextEditingController();
  var _manual = false;
  var _fingerprint = '';

  @override
  void dispose() {
    _code.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final client = widget.client;
    final wide = MediaQuery.sizeOf(context).width >= 720;
    return Scaffold(
      body: SafeArea(
        child: Align(
          alignment: Alignment.topCenter,
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 440),
            child: ListView(
              padding: EdgeInsets.fromLTRB(wide ? 48 : 24, 48, wide ? 48 : 24, 24),
              children: [
                const Text('Yoyo', style: TextStyle(fontSize: 21, fontWeight: FontWeight.w600, letterSpacing: -0.4)),
                const SizedBox(height: 12),
                const Text('扫电脑设置里的配对码。手机上的操作会在那台电脑上生效。', style: TextStyle(color: muted, fontSize: 14, height: 1.45)),
                const SizedBox(height: 28),
                if (_fingerprint.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Text(groupFingerprint(_fingerprint), style: const TextStyle(fontFamily: 'monospace', fontSize: 12.5, height: 1.4, color: muted)),
                  ),
                if (client.notice.isNotEmpty) Text(client.notice, style: const TextStyle(color: accent, fontSize: 14)),
                if (client.error.isNotEmpty) ...[
                  Text(client.error, style: const TextStyle(color: danger, fontSize: 14, height: 1.4)),
                  const SizedBox(height: 12),
                ],
                FilledButton(
                  onPressed: client.busy
                      ? null
                      : () async {
                          final raw = await Navigator.push<String>(context, MaterialPageRoute(builder: (_) => const ScanPage()));
                          if (raw == null) return;
                          setState(() => _fingerprint = PairCode.parse(raw).fingerprint);
                          await client.connect(raw);
                        },
                  child: Text(client.busy ? '正在连接' : '扫码'),
                ),
                TextButton(onPressed: () => setState(() => _manual = !_manual), child: Text(_manual ? '收起手动输入' : '输入配对码')),
                if (_manual) ...[
                  TextField(
                    controller: _code,
                    minLines: 2,
                    maxLines: 4,
                    onChanged: (_) => setState(() {}),
                    style: const TextStyle(fontFamily: 'monospace', fontSize: 12.5),
                    decoration: const InputDecoration(hintText: 'yoyo://pair?...'),
                  ),
                  const SizedBox(height: 8),
                  FilledButton(
                    onPressed: client.busy || _code.text.trim().isEmpty
                        ? null
                        : () {
                            setState(() => _fingerprint = PairCode.parse(_code.text).fingerprint);
                            client.connect(_code.text);
                          },
                    child: const Text('连接'),
                  ),
                ],
                const SizedBox(height: 8),
                TextButton(onPressed: client.busy ? null : client.useLocal, child: const Text('在这台手机上预备')),
                const SizedBox(height: 24),
                const Text('同一 Wi-Fi 名不代表互相可见。连不上时，让电脑加入手机热点。', style: TextStyle(color: muted, fontSize: 12.5, height: 1.4)),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class WorkstationShell extends StatefulWidget {
  const WorkstationShell({super.key, required this.client});
  final YoyoClient client;

  @override
  State<WorkstationShell> createState() => _WorkstationShellState();
}

class _WorkstationShellState extends State<WorkstationShell> {
  var _index = 0;

  @override
  Widget build(BuildContext context) {
    final pages = [
      SessionList(client: widget.client),
      VideoPage(client: widget.client),
      PlacePage(client: widget.client, title: '实验室', method: 'harness.get', blurb: '技能、页面和 Harness。晋升会在这台电脑上生效。'),
      SettingsPage(client: widget.client),
    ];
    final wide = MediaQuery.sizeOf(context).width >= 800;
    final body = wide
        ? Row(
            children: [
              NavigationRail(
                selectedIndex: _index,
                onDestinationSelected: (i) => setState(() => _index = i),
                labelType: NavigationRailLabelType.all,
                destinations: const [
                  NavigationRailDestination(icon: Icon(Icons.chat_bubble_outline), label: Text('会话')),
                  NavigationRailDestination(icon: Icon(Icons.movie_outlined), label: Text('视频')),
                  NavigationRailDestination(icon: Icon(Icons.science_outlined), label: Text('实验室')),
                  NavigationRailDestination(icon: Icon(Icons.tune), label: Text('设置')),
                ],
              ),
              const VerticalDivider(width: 1),
              Expanded(child: pages[_index]),
            ],
          )
        : pages[_index];
    return Scaffold(
      body: Column(
        children: [
          if (!widget.client.linked) _ReconnectBar(client: widget.client),
          Expanded(child: body),
        ],
      ),
      bottomNavigationBar: wide
          ? null
          : NavigationBar(
              selectedIndex: _index,
              onDestinationSelected: (i) => setState(() => _index = i),
              destinations: const [
                NavigationDestination(icon: Icon(Icons.chat_bubble_outline), label: '会话'),
                NavigationDestination(icon: Icon(Icons.movie_outlined), label: '视频'),
                NavigationDestination(icon: Icon(Icons.science_outlined), label: '实验室'),
                NavigationDestination(icon: Icon(Icons.tune), label: '设置'),
              ],
            ),
    );
  }
}

class _ReconnectBar extends StatelessWidget {
  const _ReconnectBar({required this.client});
  final YoyoClient client;

  @override
  Widget build(BuildContext context) {
    final text = client.notice.isNotEmpty
        ? client.notice
        : (client.error.isNotEmpty ? client.error : '连接断了。正在用这把钥匙回到电脑，当前页面会留着。');
    return Material(
      color: stoneRaised,
      child: SafeArea(
        bottom: false,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          child: Text(text, style: const TextStyle(color: accent, fontSize: 13)),
        ),
      ),
    );
  }
}

class SessionList extends StatelessWidget {
  const SessionList({super.key, required this.client});
  final YoyoClient client;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('会话'),
        actions: [
          TextButton(
            onPressed: () => Navigator.push(
              context,
              MaterialPageRoute(
                builder: (_) => SearchPage(
                  client: client,
                  onOpen: (context, row) {
                    Navigator.push(
                      context,
                      MaterialPageRoute(
                        builder: (_) => ThreadPage(
                          client: client,
                          id: '${row['id']}',
                          title: '${row['title'] ?? '未命名'}',
                          model: '${row['model'] ?? ''}',
                          workspace: '${row['workspace'] ?? ''}',
                        ),
                      ),
                    );
                  },
                ),
              ),
            ),
            child: const Text('搜索'),
          ),
          TextButton(
            onPressed: () async {
              try {
                final created = await client.call('thread.start', {'channel': 'agent'});
                await client.refresh();
                if (!context.mounted) return;
                final id = created is Map ? '${created['id']}' : '';
                if (id.isEmpty) return;
                await Navigator.push(context, MaterialPageRoute(builder: (_) => ThreadPage(client: client, id: id, title: '${created['title'] ?? '新的对话'}', model: '${created['model'] ?? ''}', workspace: '${created['workspace'] ?? ''}')));
              } on WorkstationException catch (e) {
                if (context.mounted) _toast(context, e.message);
              }
            },
            child: const Text('新建'),
          ),
        ],
      ),
      body: client.sessions.isEmpty
          ? const Center(
              child: Padding(
                padding: EdgeInsets.all(32),
                child: Text('还没有对话。新建一条，它会记在当前目的地上。', textAlign: TextAlign.center, style: TextStyle(color: muted, fontSize: 14)),
              ),
            )
          : RefreshIndicator(
              color: accent,
              backgroundColor: stoneRaised,
              onRefresh: client.refresh,
              child: ListView.separated(
                itemCount: client.sessions.length,
                separatorBuilder: (_, __) => const Divider(height: 1),
                itemBuilder: (context, i) {
                  final row = client.sessions[i];
                  final id = '${row['id']}';
                  final title = '${row['title'] ?? '未命名'}';
                  final model = '${row['model'] ?? ''}';
                  final preview = '${row['preview'] ?? ''}';
                  final meta = [if (model.isNotEmpty) model, if (preview.isNotEmpty) preview].join(' · ');
                  final live = client.running.contains(id);
                  return ListTile(
                    minVerticalPadding: 14,
                    minLeadingWidth: 12,
                    horizontalTitleGap: 8,
                    leading: Padding(
                      padding: const EdgeInsets.only(top: 6),
                      child: SizedBox(
                        width: 8,
                        height: 8,
                        child: DecoratedBox(
                          decoration: BoxDecoration(color: live ? accent : Colors.transparent, shape: BoxShape.circle),
                        ),
                      ),
                    ),
                    title: Text(title, maxLines: 1, overflow: TextOverflow.ellipsis),
                    subtitle: meta.isEmpty ? null : Text(meta, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(color: muted, fontSize: 12.5)),
                    onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => ThreadPage(client: client, id: id, title: title, model: model, workspace: '${row['workspace'] ?? ''}'))),
                    onLongPress: () => _sessionMenu(context, client, id, title),
                  );
                },
              ),
            ),
    );
  }
}

class ThreadPage extends StatefulWidget {
  const ThreadPage({super.key, required this.client, required this.id, required this.title, this.model = '', this.workspace = ''});
  final YoyoClient client;
  final String id;
  final String title;
  final String model;
  final String workspace;

  @override
  State<ThreadPage> createState() => _ThreadPageState();
}

class _ThreadPageState extends State<ThreadPage> {
  final _input = TextEditingController();
  late final TextEditingController _model = TextEditingController(text: widget.model);
  var _events = <Map<String, dynamic>>[];
  var _live = '';
  var _sending = false;
  String _error = '';
  StreamSubscription<Map<String, dynamic>>? _bus;

  @override
  void initState() {
    super.initState();
    _load();
    _bus = widget.client.port?.events.listen(_onLive);
  }

  @override
  void dispose() {
    _bus?.cancel();
    _input.dispose();
    _model.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final raw = await widget.client.call('trace.get', {'session': widget.id});
      if (!mounted) return;
      setState(() {
        _events = _eventList(raw);
        _live = '';
        _error = '';
        if (raw is Map && '${raw['model'] ?? ''}'.isNotEmpty && _model.text.isEmpty) {
          _model.text = '${raw['model']}';
        }
      });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  void _onLive(Map<String, dynamic> frame) {
    if (frame['type'] != 'live') return;
    final notice = frame['event'];
    if (notice is! Map || '${notice['session_id']}' != widget.id) return;
    final piece = _noticeText(notice);
    final type = '${notice['type']}';
    final done = type == 'turn_end' || type == 'error';
    if (!mounted) return;
    setState(() {
      if (piece.isNotEmpty) _live = '$_live$piece';
      if (done) _sending = false;
    });
    if (done) {
      unawaited(widget.client.hidePresence());
      unawaited(_load());
    }
  }

  Future<void> _send() async {
    final text = _input.text.trim();
    if (text.isEmpty || _sending) return;
    setState(() => _sending = true);
    try {
      final result = await widget.client.call('turn.start', {'session': widget.id, 'text': text, 'plan': false});
      _input.clear();
      if (result is Map && result['local'] == true) {
        final note = '${result['text'] ?? result['note'] ?? ''}';
        if (mounted && note.isNotEmpty) {
          setState(() => _events = [..._events, {'type': '本地', 'summary': note}]);
        }
        return;
      }
      await widget.client.showPresence('正在这台电脑上工作');
      await _load();
    } on WorkstationException catch (e) {
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.title, maxLines: 1, overflow: TextOverflow.ellipsis),
        actions: [
          TextButton(
            onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => QueuePage(client: widget.client, session: widget.id))),
            child: const Text('队列'),
          ),
          TextButton(
            onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => HunkPage(client: widget.client, workspace: widget.workspace))),
            child: const Text('改动'),
          ),
          TextButton(
            onPressed: () async {
              final next = await pickModel(context, widget.client, _model.text);
              if (next == null || next.trim().isEmpty) return;
              try {
                await widget.client.call('thread.model.set', {'session': widget.id, 'model': next.trim()});
                _model.text = next.trim();
                if (context.mounted) _toast(context, '已写到电脑，下一轮生效');
              } on WorkstationException catch (e) {
                if (context.mounted) _toast(context, e.message);
              }
            },
            child: const Text('模型'),
          ),
          TextButton(
            onPressed: () async {
              try {
                await widget.client.call('turn.stop', {'session': widget.id});
                await widget.client.hidePresence();
                await _load();
              } on WorkstationException catch (e) {
                if (context.mounted) _toast(context, e.message);
              }
            },
            child: const Text('停止'),
          ),
        ],
      ),
      body: Column(
        children: [
          if (_error.isNotEmpty) Padding(padding: const EdgeInsets.fromLTRB(16, 8, 16, 0), child: Text(_error, style: const TextStyle(color: danger, fontSize: 13))),
          Expanded(
            child: _events.isEmpty && _live.isEmpty
                ? const Center(child: Text('这一条还没有记录。', style: TextStyle(color: muted)))
                : ListView.builder(
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    itemCount: _events.length + (_live.isEmpty ? 0 : 1),
                    itemBuilder: (context, i) {
                      if (i == _events.length) {
                        return Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 7),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              const Text('正在写', style: TextStyle(color: accent, fontSize: 11.5)),
                              Text(_live, style: const TextStyle(fontSize: 14, height: 1.45)),
                            ],
                          ),
                        );
                      }
                      final ev = _events[i];
                      final type = '${ev['type'] ?? ev['Type'] ?? ''}';
                      final text = _eventText(ev);
                      final live = type.contains('delta') || type == 'turn_start';
                      return Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 7),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(type.isEmpty ? '记录' : type, style: TextStyle(color: live ? accent : muted, fontSize: 11.5)),
                            if (text.isNotEmpty) Text(text, style: const TextStyle(fontSize: 14, height: 1.45)),
                          ],
                        ),
                      );
                    },
                  ),
          ),
          SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _input,
                      minLines: 1,
                      maxLines: 5,
                      textInputAction: TextInputAction.send,
                      onSubmitted: (_) => _send(),
                      decoration: const InputDecoration(hintText: '发给这台工作站'),
                    ),
                  ),
                  const SizedBox(width: 8),
                  FilledButton(onPressed: _sending ? null : _send, child: const Text('发送')),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class PlacePage extends StatefulWidget {
  const PlacePage({super.key, required this.client, required this.title, required this.method, required this.blurb});
  final YoyoClient client;
  final String title;
  final String method;
  final String blurb;

  @override
  State<PlacePage> createState() => _PlacePageState();
}

class _PlacePageState extends State<PlacePage> {
  dynamic _value;
  String _error = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final value = await widget.client.call(widget.method, {});
      if (mounted) setState(() { _value = value; _error = ''; });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    final actions = widget.title == '视频'
        ? const [('连接', 'connections.list', false), ('模板', 'video.templates', false)]
        : const [('技能包', 'packs.list', false), ('页面', 'pages.list', false)];
    return Scaffold(
      appBar: AppBar(title: Text(widget.title), actions: [TextButton(onPressed: _load, child: const Text('刷新'))]),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
        children: [
          Text(widget.blurb, style: const TextStyle(color: muted, fontSize: 13, height: 1.4)),
          const SizedBox(height: 12),
          if (_error.isNotEmpty) Text(_error, style: const TextStyle(color: danger, fontSize: 13)),
          Wrap(
            spacing: 8,
            runSpacing: 4,
            children: [
              for (final action in actions)
                TextButton(onPressed: () => _run(action.$2, false), child: Text(action.$1)),
              if (widget.title == '实验室') ...[
                TextButton(onPressed: () => _checkout(context), child: const Text('用这个版本')),
                TextButton(onPressed: () => _rollback(context), child: const Text('退回')),
                TextButton(
                  onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => BrowsePage(client: widget.client, title: '日志', method: 'logs.tail', params: const {'limit': 80}))),
                  child: const Text('日志'),
                ),
                TextButton(onPressed: () => _run('evolve.last', false), child: const Text('这次进化')),
                TextButton(
                  onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => BrowsePage(client: widget.client, title: '更新', method: 'update.status'))),
                  child: const Text('更新'),
                ),
                TextButton(
                  onPressed: () async {
                    final ok = await _confirm(context, '安装更新', '确认后电脑会安装已经准备好的更新。');
                    if (ok == true) await _run('update.apply', true);
                  },
                  child: const Text('安装更新'),
                ),
              ],
            ],
          ),
          const SizedBox(height: 12),
          ValueBlock(value: _value),
        ],
      ),
    );
  }

  Future<void> _checkout(BuildContext context) async {
    try {
      final raw = await widget.client.call('harness.lineage', {});
      if (!context.mounted) return;
      final nodes = raw is Map ? _asList(raw['nodes']) : <Map<String, dynamic>>[];
      final active = raw is Map ? '${raw['active'] ?? ''}' : '';
      final hash = await showModalBottomSheet<String>(
        context: context,
        backgroundColor: stoneRaised,
        builder: (context) => SafeArea(
          child: nodes.isEmpty
              ? const Padding(padding: EdgeInsets.all(24), child: Text('还没有可切换的版本。', style: TextStyle(color: muted)))
              : ListView(
                  children: [
                    for (final node in nodes)
                      Builder(builder: (context) {
                        final hash = '${node['hash'] ?? ''}';
                        final short = hash.length <= 12 ? hash : hash.substring(0, 12);
                        return ListTile(
                          title: Text('${node['note'] ?? node['id'] ?? short}', maxLines: 1, overflow: TextOverflow.ellipsis),
                          subtitle: Text(short, style: const TextStyle(fontFamily: 'monospace', fontSize: 12, color: muted)),
                          trailing: hash == active ? const Text('当前', style: TextStyle(color: accent, fontSize: 12)) : null,
                          onTap: () => Navigator.pop(context, hash),
                        );
                      }),
                  ],
                ),
        ),
      );
      if (hash == null || hash.isEmpty || !context.mounted) return;
      final ok = await _confirm(context, '用这个版本', '确认后电脑会切到这个 Harness。');
      if (ok != true) return;
      final value = await widget.client.call('harness.checkout', {'hash': hash, 'l3': true}, confirm: true);
      if (mounted) setState(() { _value = value; _error = ''; });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  Future<void> _rollback(BuildContext context) async {
    final ok = await _confirm(context, '退回', '确认后电脑会退回上一个 Harness。');
    if (ok != true) return;
    await _run('harness.rollback', true);
  }

  Future<void> _run(String method, bool confirm) async {
    try {
      final value = await widget.client.call(method, {}, confirm: confirm);
      if (mounted) setState(() { _value = value; _error = ''; });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }
}

class SettingsPage extends StatelessWidget {
  const SettingsPage({super.key, required this.client});
  final YoyoClient client;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('设置')),
      body: ListView(
        children: [
          ListTile(
            title: Text(client.where),
            subtitle: Text(client.linked ? '操作写回这里' : '未连接', style: const TextStyle(color: muted)),
          ),
          const Padding(
            padding: EdgeInsets.fromLTRB(16, 0, 16, 8),
            child: Text('切到后台或换了网络后，连接会断。回到这里会用这把钥匙重新接上，不用再扫码。iOS 在后台留不住这条连接。', style: TextStyle(color: muted, fontSize: 12.5, height: 1.4)),
          ),
          const Divider(height: 1),
          ListTile(
            title: const Text('断开'),
            onTap: client.disconnect,
          ),
          ListTile(
            title: const Text('审批'),
            subtitle: const Text('一次、本会话、永远，或拒绝', style: TextStyle(color: muted, fontSize: 12.5)),
            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => ApprovalPage(client: client))),
          ),
          ListTile(
            title: const Text('供应商'),
            subtitle: const Text('密钥只送到电脑，手机不留副本', style: TextStyle(color: muted, fontSize: 12.5)),
            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => ProvidersPage(client: client))),
          ),
          ListTile(
            title: const Text('写入密钥'),
            subtitle: const Text('只送到电脑的 Vault，不留在手机上', style: TextStyle(color: muted, fontSize: 12.5)),
            onTap: () async {
              final key = await _ask(context, '密钥', '确认后写入电脑的 Vault', '');
              if (key == null || key.trim().isEmpty) return;
              try {
                await client.call('key.set', {'value': key.trim()}, confirm: true);
                if (context.mounted) _toast(context, '已写入电脑');
              } on WorkstationException catch (e) {
                if (context.mounted) _toast(context, e.message);
              }
            },
          ),
          ListTile(
            title: const Text('暂停工作站'),
            onTap: () async {
              try {
                await client.call('app.pause.set', {'paused': true});
                if (context.mounted) _toast(context, '电脑已暂停');
              } on WorkstationException catch (e) {
                if (context.mounted) _toast(context, e.message);
              }
            },
          ),
          ListTile(
            title: const Text('继续工作站'),
            onTap: () async {
              try {
                await client.call('app.pause.set', {'paused': false});
                if (context.mounted) _toast(context, '电脑已继续');
              } on WorkstationException catch (e) {
                if (context.mounted) _toast(context, e.message);
              }
            },
          ),
          ListTile(
            title: const Text('连接'),
            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => BrowsePage(client: client, title: '连接', method: 'connections.list', blurb: '模型、搜索和视频供应商都记在这台电脑上。'))),
          ),
          ListTile(
            title: const Text('皮肤'),
            subtitle: const Text('点一条就在电脑上启用', style: TextStyle(color: muted, fontSize: 12.5)),
            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => BrowsePage(client: client, title: '皮肤', method: 'skins.list', activate: 'skins.activate'))),
          ),
          ListTile(
            title: const Text('收件箱'),
            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => BrowsePage(client: client, title: '收件箱', method: 'inbox.list'))),
          ),
          ListTile(
            title: const Text('已配对的手机'),
            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => DevicesPage(client: client))),
          ),
          ListTile(
            title: const Text('调用工作站'),
            subtitle: const Text('方法表里的其余操作', style: TextStyle(color: muted, fontSize: 12.5)),
            onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => InvokePage(client: client))),
          ),
          if (client.error.isNotEmpty)
            Padding(padding: const EdgeInsets.all(16), child: Text(client.error, style: const TextStyle(color: danger))),
        ],
      ),
    );
  }
}

class ApprovalPage extends StatefulWidget {
  const ApprovalPage({super.key, required this.client});
  final YoyoClient client;

  @override
  State<ApprovalPage> createState() => _ApprovalPageState();
}

class _ApprovalPageState extends State<ApprovalPage> {
  var _items = <Map<String, dynamic>>[];
  String _error = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final raw = await widget.client.call('approvals.list', {});
      setState(() { _items = _asList(raw); _error = ''; });
    } on WorkstationException catch (e) {
      setState(() => _error = e.message);
    }
  }

  Future<void> _decide(String id, String decision) async {
    try {
      await widget.client.call('approvals.resolve', {'id': id, 'decision': decision});
      await _load();
    } on WorkstationException catch (e) {
      setState(() => _error = e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('审批')),
      body: _items.isEmpty
          ? Center(child: Text(_error.isEmpty ? '没有等待的决定。' : _error, style: const TextStyle(color: muted)))
          : ListView.separated(
              itemCount: _items.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, i) {
                final item = _items[i];
                final id = '${item['id'] ?? item['ID'] ?? ''}';
                final request = item['request'];
                final action = request is Map ? '${request['action'] ?? ''}' : '';
                final command = request is Map ? '${request['command'] ?? request['path'] ?? ''}' : '';
                final title = action.isNotEmpty ? action : '${item['action'] ?? item['command'] ?? id}';
                return Padding(
                  padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(title, style: const TextStyle(fontSize: 14)),
                      if (command.isNotEmpty) Text(command, style: const TextStyle(color: muted, fontSize: 12.5)),
                      const SizedBox(height: 8),
                      Wrap(
                        spacing: 8,
                        children: [
                          FilledButton(onPressed: () => _decide(id, 'once'), child: const Text('允许一次')),
                          TextButton(onPressed: () => _decide(id, 'deny'), child: const Text('拒绝')),
                          TextButton(onPressed: () => _decide(id, 'session'), child: const Text('本会话')),
                          TextButton(onPressed: () => _decide(id, 'always'), child: const Text('永远')),
                        ],
                      ),
                    ],
                  ),
                );
              },
            ),
    );
  }
}

class InvokePage extends StatefulWidget {
  const InvokePage({super.key, required this.client});
  final YoyoClient client;

  @override
  State<InvokePage> createState() => _InvokePageState();
}

class _InvokePageState extends State<InvokePage> {
  final _method = TextEditingController(text: 'health');
  final _params = TextEditingController(text: '{}');
  var _confirm = false;
  dynamic _result;
  String _error = '';

  @override
  void dispose() {
    _method.dispose();
    _params.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('调用工作站')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const Text('日常操作用前面的页面。方法表覆盖电脑上的全部调用，新增的方法也会出现在这里。', style: TextStyle(color: muted, fontSize: 13, height: 1.4)),
          const SizedBox(height: 12),
          TextButton(
            onPressed: () async {
              final picked = await Navigator.push<RemoteMethod>(context, MaterialPageRoute(builder: (_) => const MethodCatalogPage()));
              if (picked == null) return;
              _method.text = picked.name;
              _params.text = jsonEncode({for (final field in picked.fields) field: ''});
              setState(() {
                _confirm = false;
                _error = picked.denied ? '这一步只能在电脑上做。' : '';
              });
            },
            child: const Text('从方法表选择'),
          ),
          TextField(controller: _method, decoration: const InputDecoration(hintText: '方法，例如 idol.call')),
          const SizedBox(height: 8),
          TextField(controller: _params, minLines: 3, maxLines: 8, style: const TextStyle(fontFamily: 'monospace', fontSize: 12.5), decoration: const InputDecoration(hintText: '{}')),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('我已确认这是不可逆操作'),
            value: _confirm,
            onChanged: (v) => setState(() => _confirm = v),
          ),
          FilledButton(
            onPressed: () async {
              try {
                final decoded = jsonDecode(_params.text);
                if (decoded is! Map) throw const FormatException('参数要是对象');
                final known = methodByName(_method.text.trim());
                if (known != null && known.denied) {
                  setState(() => _error = '这一步只能在电脑上做。');
                  return;
                }
                if (known != null && known.confirm && !_confirm) {
                  setState(() => _error = '先打开确认，再执行这一步。');
                  return;
                }
                final params = decoded.map((k, v) => MapEntry('$k', v));
                final value = await widget.client.call(_method.text.trim(), params, confirm: _confirm);
                setState(() { _result = value; _error = ''; });
              } on WorkstationException catch (e) {
                setState(() => _error = e.message);
              } catch (e) {
                setState(() => _error = '参数不是 JSON 对象');
              }
            },
            child: const Text('执行'),
          ),
          if (_error.isNotEmpty) Padding(padding: const EdgeInsets.only(top: 12), child: Text(_error, style: const TextStyle(color: danger))),
          const SizedBox(height: 12),
          ValueBlock(value: _result),
        ],
      ),
    );
  }
}

class ValueBlock extends StatelessWidget {
  const ValueBlock({super.key, required this.value});
  final dynamic value;

  @override
  Widget build(BuildContext context) {
    if (value == null) return const SizedBox.shrink();
    if (value is Map) {
      final entries = value.entries.toList();
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (final entry in entries)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 6),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  SizedBox(width: 110, child: Text('${entry.key}', style: const TextStyle(color: muted, fontSize: 12.5))),
                  Expanded(child: Text(_short(entry.value), style: const TextStyle(fontSize: 13, height: 1.35))),
                ],
              ),
            ),
        ],
      );
    }
    return Text(_short(value), style: const TextStyle(fontFamily: 'monospace', fontSize: 12, height: 1.4));
  }
}

String _short(dynamic value) {
  if (value is Map || value is List) {
    final text = jsonEncode(value);
    return text.length > 600 ? '${text.substring(0, 600)}…' : text;
  }
  return '$value';
}

List<Map<String, dynamic>> _asList(dynamic raw) {
  if (raw is List) {
    return raw.whereType<Map>().map((m) => m.map((k, v) => MapEntry('$k', v))).toList();
  }
  return [];
}

List<Map<String, dynamic>> _eventList(dynamic raw) {
  if (raw is List) return _asList(raw);
  if (raw is Map) {
    for (final key in ['events', 'items', 'turns']) {
      if (raw[key] is List) return _asList(raw[key]);
    }
  }
  return [];
}

List<String> _modelIds(dynamic catalog) {
  final out = <String>[];
  if (catalog is! Map || catalog['providers'] is! Map) return out;
  final providers = catalog['providers'] as Map;
  for (final group in providers.values) {
    if (group is! Map) continue;
    for (final entry in group.values) {
      if (entry is! Map) continue;
      final model = entry['model'];
      final id = model is Map ? model['id'] : entry['id'];
      if (id != null && '$id'.isNotEmpty) out.add('$id');
    }
  }
  return out;
}

String _eventText(Map<String, dynamic> ev) {
  final summary = ev['summary'] ?? ev['Summary'];
  if (summary != null && '$summary'.isNotEmpty) return '$summary';
  final detail = ev['detail'] ?? ev['Detail'];
  if (detail != null && '$detail'.isNotEmpty) return '$detail';
  final payload = ev['payload'] ?? ev['Payload'];
  if (payload is Map) {
    for (final key in ['text', 'delta', 'message', 'content', 'command']) {
      final value = payload[key];
      if (value != null && '$value'.isNotEmpty) return '$value';
    }
  }
  if (payload is String) return payload;
  final direct = ev['text'] ?? ev['message'];
  return direct == null ? '' : '$direct';
}

String _noticeText(Map notice) {
  final frame = notice['frame'];
  if (frame is Map && frame['delta'] != null && '${frame['delta']}'.isNotEmpty) return '${frame['delta']}';
  final event = notice['event'];
  if (event is Map) return _eventText(event.map((k, v) => MapEntry('$k', v)));
  return '';
}

Future<bool?> _confirm(BuildContext context, String title, String body) {
  return showDialog<bool>(
    context: context,
    builder: (context) => AlertDialog(
      title: Text(title),
      content: Text(body),
      actions: [
        TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
        FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('确认')),
      ],
    ),
  );
}

Future<void> _sessionMenu(BuildContext context, YoyoClient client, String id, String title) async {
  final action = await showModalBottomSheet<String>(
    context: context,
    backgroundColor: stoneRaised,
    builder: (context) => SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ListTile(title: const Text('重命名'), onTap: () => Navigator.pop(context, 'rename')),
          ListTile(title: const Text('置顶'), onTap: () => Navigator.pop(context, 'pin')),
          ListTile(title: const Text('分叉'), onTap: () => Navigator.pop(context, 'fork')),
          ListTile(title: const Text('归档'), onTap: () => Navigator.pop(context, 'archive')),
          ListTile(title: const Text('删除'), onTap: () => Navigator.pop(context, 'delete')),
        ],
      ),
    ),
  );
  if (!context.mounted || action == null) return;
  if (action == 'rename') {
    final next = await _ask(context, '重命名', title, title);
    if (next == null || next.trim().isEmpty) return;
    try {
      await client.call('thread.rename', {'session': id, 'title': next.trim()});
      await client.refresh();
    } on WorkstationException catch (e) {
      if (context.mounted) _toast(context, e.message);
    }
    return;
  }
  if (action == 'pin') {
    try {
      await client.call('thread.pin', {'session': id, 'pinned': true});
      await client.refresh();
    } on WorkstationException catch (e) {
      if (context.mounted) _toast(context, e.message);
    }
    return;
  }
  if (action == 'archive') {
    try {
      await client.call('thread.archive', {'session': id, 'archived': true});
      await client.refresh();
    } on WorkstationException catch (e) {
      if (context.mounted) _toast(context, e.message);
    }
    return;
  }
  if (action == 'fork') {
    try {
      final created = await client.call('thread.fork', {'session': id});
      await client.refresh();
      if (!context.mounted || created is! Map) return;
      final nextID = '${created['id'] ?? ''}';
      if (nextID.isEmpty) {
        final note = '${created['note'] ?? created['text'] ?? ''}';
        if (note.isNotEmpty && context.mounted) _toast(context, note);
        return;
      }
      await Navigator.push(
        context,
        MaterialPageRoute(
          builder: (_) => ThreadPage(
            client: client,
            id: nextID,
            title: '${created['title'] ?? title}',
            model: '${created['model'] ?? ''}',
            workspace: '${created['workspace'] ?? ''}',
          ),
        ),
      );
    } on WorkstationException catch (e) {
      if (context.mounted) _toast(context, e.message);
    }
    return;
  }
  final ok = await _confirm(context, '删除这条对话', '确认后电脑上的这条对话会被删除。');
  if (ok != true) return;
  try {
    await client.call('thread.delete', {'session': id}, confirm: true);
    await client.refresh();
  } on WorkstationException catch (e) {
    if (context.mounted) _toast(context, e.message);
  }
}

Future<String?> pickModel(BuildContext context, YoyoClient client, String current) async {
  final models = <String>{};
  try {
    models.addAll(_modelIds(await client.call('models.catalog', {})));
  } catch (_) {}
  try {
    final providers = await client.call('provider.list', {});
    if (providers is List) {
      for (final provider in providers.whereType<Map>()) {
        final model = provider['model'];
        if (model != null && '$model'.isNotEmpty) models.add('$model');
        final many = provider['models'];
        if (many is List) {
          for (final item in many) {
            if ('$item'.isNotEmpty) models.add('$item');
          }
        }
      }
    }
  } catch (_) {}
  if (!context.mounted) return null;
  final ordered = models.toList()..sort();
  return showModalBottomSheet<String>(
    context: context,
    backgroundColor: stoneRaised,
    isScrollControlled: true,
    builder: (context) => _ModelSheet(current: current, models: ordered),
  );
}

class _ModelSheet extends StatefulWidget {
  const _ModelSheet({required this.current, required this.models});
  final String current;
  final List<String> models;

  @override
  State<_ModelSheet> createState() => _ModelSheetState();
}

class _ModelSheetState extends State<_ModelSheet> {
  late final TextEditingController _custom = TextEditingController(text: widget.current);

  @override
  void dispose() {
    _custom.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final height = MediaQuery.sizeOf(context).height * 0.72;
    return SafeArea(
      child: SizedBox(
        height: height,
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
              child: Row(
                children: [
                  Expanded(child: TextField(controller: _custom, decoration: const InputDecoration(hintText: '模型，下一轮生效'))),
                  const SizedBox(width: 8),
                  FilledButton(onPressed: () => Navigator.pop(context, _custom.text.trim()), child: const Text('使用')),
                ],
              ),
            ),
            const Divider(height: 1),
            Expanded(
              child: widget.models.isEmpty
                  ? const Center(child: Text('目录暂时取不到。直接写下模型名。', style: TextStyle(color: muted)))
                  : ListView.builder(
                      itemCount: widget.models.length,
                      itemBuilder: (context, i) {
                        final model = widget.models[i];
                        return ListTile(
                          title: Text(model, maxLines: 1, overflow: TextOverflow.ellipsis),
                          trailing: model == widget.current ? const Text('当前', style: TextStyle(color: accent, fontSize: 12)) : null,
                          onTap: () => Navigator.pop(context, model),
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

class ProvidersPage extends StatefulWidget {
  const ProvidersPage({super.key, required this.client});
  final YoyoClient client;

  @override
  State<ProvidersPage> createState() => _ProvidersPageState();
}

class _ProvidersPageState extends State<ProvidersPage> {
  var _items = <Map<String, dynamic>>[];
  String _error = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final raw = await widget.client.call('provider.list', {});
      if (mounted) setState(() { _items = _asList(raw); _error = ''; });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('供应商'),
        actions: [
          TextButton(
            onPressed: () async {
              final name = await _ask(context, '名称', '显示名', '');
              if (name == null || name.trim().isEmpty || !context.mounted) return;
              final endpoint = await _ask(context, '地址', 'https://', '');
              if (endpoint == null || !context.mounted) return;
              final model = await _ask(context, '模型', '默认模型', '');
              if (model == null || !context.mounted) return;
              final secret = await _ask(context, '密钥', '可留空。确认后只送到电脑', '');
              if (secret == null || !context.mounted) return;
              final ok = await _confirm(context, '写入供应商', '确认后电脑会保存这个供应商。');
              if (ok != true) return;
              try {
                await widget.client.call('provider.upsert', {
                  'provider': {'name': name.trim(), 'endpoint': endpoint.trim(), 'model': model.trim()},
                  'api_key': secret.trim(),
                }, confirm: true);
                await _load();
              } on WorkstationException catch (e) {
                if (mounted) setState(() => _error = e.message);
              }
            },
            child: const Text('添加'),
          ),
        ],
      ),
      body: _items.isEmpty
          ? Center(child: Text(_error.isEmpty ? '还没有供应商。' : _error, style: const TextStyle(color: muted)))
          : ListView.separated(
              itemCount: _items.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, i) {
                final item = _items[i];
                final id = '${item['id'] ?? ''}';
                final name = '${item['name'] ?? id}';
                final model = '${item['model'] ?? ''}';
                final keyed = item['has_key'] == true ? '已有密钥' : '没有密钥';
                return ListTile(
                  title: Text(name),
                  subtitle: Text([if (model.isNotEmpty) model, keyed].join(' · '), style: const TextStyle(color: muted, fontSize: 12.5)),
                  trailing: item['is_default'] == true ? const Text('默认', style: TextStyle(color: accent, fontSize: 12)) : null,
                  onTap: () async {
                    try {
                      await widget.client.call('provider.setDefault', {'id': id});
                      await _load();
                    } on WorkstationException catch (e) {
                      if (mounted) setState(() => _error = e.message);
                    }
                  },
                  onLongPress: () async {
                    final ok = await _confirm(context, '删除 $name', '确认后电脑会去掉这个供应商。');
                    if (ok != true) return;
                    try {
                      await widget.client.call('provider.delete', {'id': id}, confirm: true);
                      await _load();
                    } on WorkstationException catch (e) {
                      if (mounted) setState(() => _error = e.message);
                    }
                  },
                );
              },
            ),
    );
  }
}

class DevicesPage extends StatefulWidget {
  const DevicesPage({super.key, required this.client});
  final YoyoClient client;

  @override
  State<DevicesPage> createState() => _DevicesPageState();
}

class _DevicesPageState extends State<DevicesPage> {
  var _items = <Map<String, dynamic>>[];
  String _error = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final raw = await widget.client.call('remote.status', {});
      final devices = raw is Map ? _asList(raw['devices']) : <Map<String, dynamic>>[];
      if (mounted) setState(() { _items = devices.where((d) => d['revoked'] != true).toList(); _error = ''; });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('已配对的手机')),
      body: _items.isEmpty
          ? Center(child: Text(_error.isEmpty ? '还没有别的手机。' : _error, style: const TextStyle(color: muted)))
          : ListView.separated(
              itemCount: _items.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, i) {
                final item = _items[i];
                final id = '${item['id'] ?? ''}';
                return ListTile(
                  title: Text('${item['name'] ?? id}'),
                  subtitle: Text(id, style: const TextStyle(fontFamily: 'monospace', fontSize: 12, color: muted)),
                  trailing: TextButton(
                    onPressed: () async {
                      final ok = await _confirm(context, '移除这台手机', '移除后它要重新扫码。');
                      if (ok != true) return;
                      try {
                        await widget.client.call('remote.revoke', {'id': id});
                        await _load();
                      } on WorkstationException catch (e) {
                        if (mounted) setState(() => _error = e.message);
                      }
                    },
                    child: const Text('移除'),
                  ),
                );
              },
            ),
    );
  }
}

Future<String?> _ask(BuildContext context, String title, String hint, String initial) {
  final controller = TextEditingController(text: initial);
  return showDialog<String>(
    context: context,
    builder: (context) => AlertDialog(
      title: Text(title),
      content: TextField(controller: controller, decoration: InputDecoration(hintText: hint), autofocus: true),
      actions: [
        TextButton(onPressed: () => Navigator.pop(context), child: const Text('取消')),
        FilledButton(onPressed: () => Navigator.pop(context, controller.text), child: const Text('写入')),
      ],
    ),
  );
}

void _toast(BuildContext context, String text) {
  ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(text), behavior: SnackBarBehavior.floating));
}
