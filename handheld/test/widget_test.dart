import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';

import 'package:yoyo_handheld/client.dart';
import 'package:yoyo_handheld/methods.g.dart';
import 'package:yoyo_handheld/shell.dart';
import 'package:yoyo_handheld/surfaces.dart';
import 'package:yoyo_handheld/workstation.dart';

void main() {
  testWidgets('local destination can start a thread', (tester) async {
    final client = YoyoClient(initial: LocalWorkstation());
    await client.refresh();
    await tester.pumpWidget(YoyoApp(client: client));
    await tester.pumpAndSettle();

    expect(find.text('会话'), findsWidgets);
    await tester.tap(find.text('新建'));
    await tester.pumpAndSettle();

    expect(find.text('新的对话'), findsWidgets);
    expect(find.text('发给这台工作站'), findsOneWidget);
  });

  test('model change on the local port stays on that port', () async {
    final local = LocalWorkstation();
    final created = await local.call('thread.start', {'channel': 'agent'}) as Map;
    await local.call('thread.model.set', {'session': created['id'], 'model': 'gpt-handheld'});
    final list = await local.call('thread.list', {}) as List;
    expect(list.single['model'], 'gpt-handheld');
  });

  test('the method table is reachable and a local key is not stored', () async {
    expect(methodByName('thread.model.set')?.confirm, isFalse);
    expect(methodByName('key.set')?.confirm, isTrue);
    expect(methodByName('key.set')?.fields, contains('value'));
    expect(methodByName('remote.pair.begin')?.denied, isTrue);
    expect(remoteMethods.length, greaterThan(200));
    final out = await LocalWorkstation().call('key.set', {'value': 'sk-secret'}, confirm: true);
    expect(jsonEncode(out), isNot(contains('sk-secret')));
  });
}
