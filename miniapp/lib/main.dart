import 'package:flutter/material.dart';
import 'config/api.dart';
import 'pages/home.dart';
import 'pages/orders.dart';
import 'pages/messages.dart';
import 'pages/profile.dart';
import 'pages/login.dart';

void main() {
  runApp(const WDOSApp());
}

class WDOSApp extends StatelessWidget {
  const WDOSApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'WDOS 工单系统',
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF1989FA),
        useMaterial3: true,
      ),
      home: const AuthGate(),
      debugShowCheckedModeBanner: false,
    );
  }
}

/// 认证网关：有 token 进首页，无 token 进登录页
class AuthGate extends StatefulWidget {
  const AuthGate({super.key});
  @override
  State<AuthGate> createState() => _AuthGateState();
}

class _AuthGateState extends State<AuthGate> {
  bool _checking = true;
  bool _loggedIn = false;

  @override
  void initState() {
    super.initState();
    _checkAuth();
  }

  Future<void> _checkAuth() async {
    final token = await ApiConfig.getToken();
    setState(() { _loggedIn = token != null; _checking = false; });
  }

  @override
  Widget build(BuildContext context) {
    if (_checking) return const Scaffold(body: Center(child: CircularProgressIndicator()));
    if (!_loggedIn) return const LoginPage();
    return const MainTabs();
  }
}

/// 底部 Tab 导航
class MainTabs extends StatefulWidget {
  const MainTabs({super.key});
  @override
  State<MainTabs> createState() => _MainTabsState();
}

class _MainTabsState extends State<MainTabs> {
  int _currentIndex = 0;

  final _pages = const [
    HomePage(),
    OrdersPage(),
    MessagesPage(),
    ProfilePage(),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: _pages[_currentIndex],
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex,
        onDestinationSelected: (i) => setState(() => _currentIndex = i),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.dashboard_outlined), selectedIcon: Icon(Icons.dashboard), label: '工作台'),
          NavigationDestination(icon: Icon(Icons.assignment_outlined), selectedIcon: Icon(Icons.assignment), label: '工单'),
          NavigationDestination(icon: Icon(Icons.notifications_outlined), selectedIcon: Icon(Icons.notifications), label: '消息'),
          NavigationDestination(icon: Icon(Icons.person_outlined), selectedIcon: Icon(Icons.person), label: '我的'),
        ],
      ),
    );
  }
}
