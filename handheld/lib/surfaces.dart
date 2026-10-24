import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/material.dart';

import 'client.dart';
import 'methods.g.dart';
import 'theme.dart';
import 'workstation.dart';

RemoteMethod? methodByName(String name) {
  for (final method in remoteMethods) {
    if (method.name == name) return method;
  }
  return null;
}

class MethodCatalogPage extends StatefulWidget {
  const MethodCatalogPage({super.key});

  @override
  State<MethodCatalogPage> createState() => _MethodCatalogPageState();
}

class _MethodCatalogPageState extends State<MethodCatalogPage> {
  final _query = TextEditingController();
  var _group = '全部';

  @override
  void dispose() {
    _query.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final groups = ['全部', ...{for (final method in remoteMethods) method.group}];
    final q = _query.text.trim().toLowerCase();
    final shown = remoteMethods.where((method) {
      if (_group != '全部' && method.group != _group) return false;
      if (q.isEmpty) return true;
      return method.name.contains(q);
    }).toList();
    return Scaffold(
      appBar: AppBar(title: const Text('方法表')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
            child: TextField(
              controller: _query,
              onChanged: (_) => setState(() {}),
              decoration: const InputDecoration(hintText: '搜索方法'),
            ),
          ),
          SizedBox(
            height: 48,
            child: ListView(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.symmetric(horizontal: 12),
              children: [
                for (final group in groups)
                  TextButton(
                    onPressed: () => setState(() => _group = group),
                    child: Text(group, style: TextStyle(color: _group == group ? accent : muted)),
                  ),
              ],
            ),
          ),
          const Divider(height: 1),
          Expanded(
            child: ListView.separated(
              itemCount: shown.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, i) {
                final method = shown[i];
                final note = method.denied ? '只能在电脑上' : (method.confirm ? '需要确认' : method.group);
                return ListTile(
                  title: Text(method.name, style: const TextStyle(fontFamily: 'monospace', fontSize: 13)),
                  subtitle: Text(note, style: const TextStyle(color: muted, fontSize: 12)),
                  onTap: () => Navigator.pop(context, method),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}

class VideoPage extends StatefulWidget {
  const VideoPage({super.key, required this.client});
  final YoyoClient client;

  @override
  State<VideoPage> createState() => _VideoPageState();
}

class _VideoPageState extends State<VideoPage> {
  static const _lanes = [
    ('状态', 'video.status'),
    ('短片', 'video.studio.list'),
    ('短剧', 'drama.list'),
    ('记录', 'video.history'),
  ];
  var _lane = 0;
  dynamic _value;
  String _error = '';
  final _path = TextEditingController();
  Uint8List? _preview;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _path.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final value = await widget.client.call(_lanes[_lane].$2, {});
      if (mounted) setState(() { _value = value; _error = ''; });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  Future<void> _previewPath() async {
    final path = _path.text.trim();
    if (path.isEmpty) return;
    final port = widget.client.port;
    if (port is! RemoteWorkstation) {
      setState(() => _error = '预览文件要连着电脑。生成结果留在电脑的视频目录里。');
      return;
    }
    try {
      final bytes = await port.media(path);
      if (mounted) setState(() { _preview = bytes; _error = ''; });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('视频'), actions: [TextButton(onPressed: _load, child: const Text('刷新'))]),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
        children: [
          const Text('状态、短片、短剧和生成记录都在这台电脑上跑。画布在电脑上编辑，绑定工程用方法表里的 canvas.bind。文件留在电脑的视频目录。', style: TextStyle(color: muted, fontSize: 13, height: 1.4)),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            children: [
              for (var i = 0; i < _lanes.length; i++)
                TextButton(
                  onPressed: () {
                    setState(() => _lane = i);
                    _load();
                  },
                  child: Text(_lanes[i].$1, style: TextStyle(color: _lane == i ? accent : ink)),
                ),
            ],
          ),
          if (_error.isNotEmpty) Text(_error, style: const TextStyle(color: danger, fontSize: 13, height: 1.4)),
          const SizedBox(height: 8),
          RecordView(value: _value),
          const SizedBox(height: 16),
          const Text('按电脑上的路径预览一张图', style: TextStyle(color: muted, fontSize: 12.5)),
          const SizedBox(height: 8),
          TextField(controller: _path, decoration: const InputDecoration(hintText: 'video/…png')),
          const SizedBox(height: 8),
          TextButton(onPressed: _previewPath, child: const Text('预览')),
          if (_preview != null) Padding(padding: const EdgeInsets.only(top: 12), child: Image.memory(_preview!, height: 220, fit: BoxFit.contain)),
        ],
      ),
    );
  }
}

class BrowsePage extends StatefulWidget {
  const BrowsePage({super.key, required this.client, required this.title, required this.method, this.params = const {}, this.blurb = '', this.activate});
  final YoyoClient client;
  final String title;
  final String method;
  final Map<String, dynamic> params;
  final String blurb;
  final String? activate;

  @override
  State<BrowsePage> createState() => _BrowsePageState();
}

class _BrowsePageState extends State<BrowsePage> {
  dynamic _value;
  String _error = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final value = await widget.client.call(widget.method, Map<String, dynamic>.from(widget.params));
      if (mounted) setState(() { _value = value; _error = ''; });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    final rows = _rows(_value);
    return Scaffold(
      appBar: AppBar(title: Text(widget.title), actions: [TextButton(onPressed: _load, child: const Text('刷新'))]),
      body: rows == null
          ? ListView(
              padding: const EdgeInsets.all(20),
              children: [
                if (widget.blurb.isNotEmpty) Text(widget.blurb, style: const TextStyle(color: muted, fontSize: 13, height: 1.4)),
                if (_error.isNotEmpty) Text(_error, style: const TextStyle(color: danger, fontSize: 13)),
                RecordView(value: _value),
              ],
            )
          : ListView.separated(
              itemCount: rows.isEmpty ? 1 : rows.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, i) {
                if (rows.isEmpty) {
                  return Padding(
                    padding: const EdgeInsets.all(24),
                    child: Text(_error.isEmpty ? '这里还是空的。' : _error, style: const TextStyle(color: muted)),
                  );
                }
                final row = rows[i];
                final id = '${row['id'] ?? row['ID'] ?? ''}';
                final title = '${row['name'] ?? row['title'] ?? row['file'] ?? row['text'] ?? id}';
                return ListTile(
                  title: Text(title, maxLines: 2, overflow: TextOverflow.ellipsis),
                  subtitle: id.isEmpty ? null : Text(id, style: const TextStyle(color: muted, fontSize: 12)),
                  onTap: widget.activate == null || id.isEmpty
                      ? null
                      : () async {
                          try {
                            await widget.client.call(widget.activate!, {'id': id});
                            if (context.mounted) ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('已写到电脑')));
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

class QueuePage extends StatefulWidget {
  const QueuePage({super.key, required this.client, required this.session});
  final YoyoClient client;
  final String session;

  @override
  State<QueuePage> createState() => _QueuePageState();
}

class _QueuePageState extends State<QueuePage> {
  var _items = <Map<String, dynamic>>[];
  String _error = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final raw = await widget.client.call('thread.queue.list', {'session': widget.session});
      if (mounted) setState(() { _items = _rows(raw) ?? []; _error = ''; });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('队列')),
      body: _items.isEmpty
          ? Center(child: Text(_error.isEmpty ? '没有排着的下一句。' : _error, style: const TextStyle(color: muted)))
          : ListView.separated(
              itemCount: _items.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, i) {
                final item = _items[i];
                final id = '${item['id'] ?? ''}';
                return ListTile(
                  title: Text('${item['text'] ?? ''}', maxLines: 3, overflow: TextOverflow.ellipsis),
                  trailing: TextButton(
                    onPressed: id.isEmpty
                        ? null
                        : () async {
                            try {
                              await widget.client.call('thread.queue.cancel', {'session': widget.session, 'id': id});
                              await _load();
                            } on WorkstationException catch (e) {
                              if (mounted) setState(() => _error = e.message);
                            }
                          },
                    child: const Text('取消'),
                  ),
                );
              },
            ),
    );
  }
}

class HunkPage extends StatefulWidget {
  const HunkPage({super.key, required this.client, required this.workspace});
  final YoyoClient client;
  final String workspace;

  @override
  State<HunkPage> createState() => _HunkPageState();
}

class _HunkPageState extends State<HunkPage> {
  var _items = <Map<String, dynamic>>[];
  String _error = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    if (widget.workspace.isEmpty) {
      setState(() => _error = '这条对话没有工作区。');
      return;
    }
    try {
      final raw = await widget.client.call('workspace.hunks', {'workspace': widget.workspace});
      final hunks = raw is Map ? raw['hunks'] : raw;
      if (mounted) setState(() { _items = _rows(hunks) ?? []; _error = ''; });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  Future<void> _apply(List<String> ids) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('接受这些改动'),
        content: const Text('确认后会写进电脑上的工作区。'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('确认')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await widget.client.call('workspace.apply_hunks', {'workspace': widget.workspace, 'ids': ids}, confirm: true);
      await _load();
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('工作区改动'),
        actions: [
          if (_items.isNotEmpty)
            TextButton(onPressed: () => _apply([for (final item in _items) '${item['id']}']), child: const Text('全部接受')),
        ],
      ),
      body: _items.isEmpty
          ? Center(child: Text(_error.isEmpty ? '工作区没有未接受的改动。' : _error, style: const TextStyle(color: muted)))
          : ListView.separated(
              itemCount: _items.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, i) {
                final item = _items[i];
                return ListTile(
                  title: Text('${item['file'] ?? item['id']}', maxLines: 1, overflow: TextOverflow.ellipsis),
                  subtitle: Text('${item['header'] ?? ''}', maxLines: 2, overflow: TextOverflow.ellipsis, style: const TextStyle(fontFamily: 'monospace', fontSize: 12, color: muted)),
                  onTap: () => _apply(['${item['id']}']),
                );
              },
            ),
    );
  }
}

class SearchPage extends StatefulWidget {
  const SearchPage({super.key, required this.client, required this.onOpen});
  final YoyoClient client;
  final void Function(BuildContext context, Map<String, dynamic> row) onOpen;

  @override
  State<SearchPage> createState() => _SearchPageState();
}

class _SearchPageState extends State<SearchPage> {
  final _query = TextEditingController();
  var _items = <Map<String, dynamic>>[];
  String _error = '';

  @override
  void dispose() {
    _query.dispose();
    super.dispose();
  }

  Future<void> _run() async {
    try {
      final raw = await widget.client.call('thread.search', {'query': _query.text.trim(), 'include_archived': true});
      if (mounted) setState(() { _items = _rows(raw) ?? []; _error = ''; });
    } on WorkstationException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('搜索')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: TextField(
              controller: _query,
              textInputAction: TextInputAction.search,
              onSubmitted: (_) => _run(),
              decoration: const InputDecoration(hintText: '在电脑上的对话里找'),
            ),
          ),
          if (_error.isNotEmpty) Text(_error, style: const TextStyle(color: danger)),
          Expanded(
            child: ListView.separated(
              itemCount: _items.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, i) {
                final row = _items[i];
                return ListTile(
                  title: Text('${row['title'] ?? '未命名'}'),
                  subtitle: Text('${row['preview'] ?? row['model'] ?? ''}', maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(color: muted, fontSize: 12.5)),
                  onTap: () => widget.onOpen(context, row),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}

class RecordView extends StatelessWidget {
  const RecordView({super.key, required this.value});
  final dynamic value;

  @override
  Widget build(BuildContext context) {
    if (value == null) return const SizedBox.shrink();
    if (value is Map) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (final entry in value.entries)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 6),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  SizedBox(width: 110, child: Text('${entry.key}', style: const TextStyle(color: muted, fontSize: 12.5))),
                  Expanded(child: Text(_clip(entry.value), style: const TextStyle(fontSize: 13, height: 1.35))),
                ],
              ),
            ),
        ],
      );
    }
    return Text(_clip(value), style: const TextStyle(fontFamily: 'monospace', fontSize: 12, height: 1.4));
  }
}

List<Map<String, dynamic>>? _rows(dynamic raw) {
  if (raw is List) {
    return raw.whereType<Map>().map((m) => m.map((k, v) => MapEntry('$k', v))).toList();
  }
  if (raw is Map) {
    for (final value in raw.values) {
      if (value is List && value.every((item) => item is Map)) {
        return _rows(value);
      }
    }
  }
  return null;
}

String _clip(dynamic value) {
  if (value is Map || value is List) {
    final text = jsonEncode(value);
    return text.length > 600 ? '${text.substring(0, 600)}…' : text;
  }
  return '$value';
}
