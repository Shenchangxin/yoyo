import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import 'theme.dart';

class ScanPage extends StatefulWidget {
  const ScanPage({super.key});

  @override
  State<ScanPage> createState() => _ScanPageState();
}

class _ScanPageState extends State<ScanPage> {
  var _done = false;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('扫配对码')),
      body: Stack(
        children: [
          MobileScanner(
            onDetect: (capture) {
              if (_done) return;
              final raw = capture.barcodes.isEmpty ? null : capture.barcodes.first.rawValue;
              if (raw == null || !_isPair(raw)) return;
              _done = true;
              Navigator.pop(context, raw);
            },
          ),
          const Align(
            alignment: Alignment.bottomCenter,
            child: SafeArea(
              child: Padding(
                padding: EdgeInsets.all(24),
                child: Text('把电脑上的配对码放进画面', style: TextStyle(color: ink, fontSize: 14)),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

bool _isPair(String raw) {
  final uri = Uri.tryParse(raw.trim());
  if (uri == null) return false;
  if (uri.scheme == 'yoyo') return true;
  if (uri.scheme != 'http' && uri.scheme != 'https') return false;
  return uri.queryParameters['n']?.isNotEmpty == true && uri.queryParameters['fp']?.isNotEmpty == true;
}
