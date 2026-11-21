import 'package:flutter/material.dart';
import '../core/router/app_router.dart';
import '../core/theme/app_theme.dart';

class LoveTeaApp extends StatelessWidget {
  const LoveTeaApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'LoVeTea',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.lightTheme,
      routerConfig: appRouter,
    );
  }
}
