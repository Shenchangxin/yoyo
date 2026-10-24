import 'package:flutter/material.dart';

const stone = Color(0xFF1C1C1A);
const stoneRaised = Color(0xFF222220);
const ink = Color(0xFFF4F4F0);
const muted = Color(0xFF8C8B85);
const accent = Color(0xFF5C99D6);
const danger = Color(0xFFE06B66);
const line = Color(0x17F4F4F0);

ThemeData yoyoTheme() {
  final base = ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    scaffoldBackgroundColor: stone,
    canvasColor: stone,
    splashFactory: NoSplash.splashFactory,
    highlightColor: Colors.transparent,
    dividerColor: line,
    fontFamily: 'sans-serif',
  );
  return base.copyWith(
    colorScheme: const ColorScheme.dark(
      surface: stone,
      primary: ink,
      onPrimary: stone,
      secondary: accent,
      error: danger,
      onSurface: ink,
    ),
    textTheme: base.textTheme.apply(bodyColor: ink, displayColor: ink).copyWith(
          titleLarge: const TextStyle(fontSize: 21, fontWeight: FontWeight.w600, letterSpacing: -0.4, color: ink),
          titleMedium: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600, color: ink),
          bodyMedium: const TextStyle(fontSize: 14, height: 1.45, color: ink),
          bodySmall: const TextStyle(fontSize: 12.5, height: 1.4, color: muted),
          labelLarge: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
        ),
    appBarTheme: const AppBarTheme(
      backgroundColor: stone,
      foregroundColor: ink,
      elevation: 0,
      scrolledUnderElevation: 0,
      centerTitle: false,
    ),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: stoneRaised,
      indicatorColor: const Color(0xFF32322E),
      height: 64,
      labelTextStyle: WidgetStateProperty.all(const TextStyle(fontSize: 11, fontWeight: FontWeight.w500)),
    ),
    navigationRailTheme: const NavigationRailThemeData(
      backgroundColor: stoneRaised,
      indicatorColor: Color(0xFF32322E),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: ink,
        foregroundColor: stone,
        minimumSize: const Size(48, 48),
        textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: ink,
        minimumSize: const Size(48, 48),
        textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w500),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: stoneRaised,
      hintStyle: const TextStyle(color: muted, fontSize: 14),
      contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
      border: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: line)),
      enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: line)),
      focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(8), borderSide: const BorderSide(color: accent)),
    ),
    dialogTheme: const DialogTheme(
      backgroundColor: Color(0xFF242422),
    ),
  );
}

String groupFingerprint(String fp) {
  final clean = fp.replaceAll(RegExp(r'\s'), '');
  final parts = <String>[];
  for (var i = 0; i < clean.length; i += 4) {
    final end = i + 4 > clean.length ? clean.length : i + 4;
    parts.add(clean.substring(i, end));
  }
  return parts.join(' ');
}
