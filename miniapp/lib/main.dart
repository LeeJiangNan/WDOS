import 'package:flutter/material.dart';
import 'config/api.dart';
import 'services/api.dart';
import 'pages/home.dart';
import 'pages/orders.dart';
import 'pages/messages.dart';
import 'pages/profile.dart';
import 'pages/login.dart';

void main() {
  ApiConfig.init();
  runApp(const WDOSApp());
}

class WDOSApp extends StatelessWidget {
  const WDOSApp({super.key});
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'WDOS 工单系统',
      theme: ThemeData(colorSchemeSeed: const Color(0xFF1989FA), useMaterial3: true),
      home: const AuthGate(),
      debugShowCheckedModeBanner: false,
    );
  }
}

class AuthGate extends StatefulWidget {
  const AuthGate({super.key});
  @override
  State<AuthGate> createState() => _AuthGateState();
}

class _AuthGateState extends State<AuthGate> {
  bool _checking = true;
  bool _loggedIn = false;

  @override
  void initState() { super.initState(); _checkAuth(); }

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

class MainTabs extends StatefulWidget {
  const MainTabs({super.key});
  @override
  State<MainTabs> createState() => _MainTabsState();
}

class _MainTabsState extends State<MainTabs> {
  int _currentIndex = 0;
  final _ordersKey = GlobalKey<OrdersPageState>();
  int _pendingCount = 0;

  @override
  void initState() {
    super.initState();
    _loadBadge();
  }

  Future<void> _loadBadge() async {
    try {
      final data = await ApiService.get('/work-orders/pending');
      if (mounted) setState(() => _pendingCount = (data['list'] as List?)?.length ?? 0);
    } catch (_) {}
  }

  void _switchToOrders(int tabIndex) {
    setState(() => _currentIndex = 1);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _ordersKey.currentState?.switchToTab(tabIndex);
    });
  }

  @override
  Widget build(BuildContext context) {
    // 每次重建都重新构建 _pages 以确保回调最新
    final pages = <Widget>[
      HomePage(onNavigateOrders: _switchToOrders),
      OrdersPage(key: _ordersKey),
      const MessagesPage(),
      ProfilePage(onGoToOrders: () => _switchToOrders(0)),
    ];

    return Scaffold(
      body: pages[_currentIndex],
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex,
        onDestinationSelected: (i) {
          setState(() => _currentIndex = i);
          if (i == 2) _loadBadge();
        },
        destinations: [
          const NavigationDestination(icon: Icon(Icons.dashboard_outlined), selectedIcon: Icon(Icons.dashboard), label: '工作台'),
          const NavigationDestination(icon: Icon(Icons.assignment_outlined), selectedIcon: Icon(Icons.assignment), label: '工单'),
          NavigationDestination(
            icon: Badge(label: _pendingCount > 0 ? Text('$_pendingCount') : null, isLabelVisible: _pendingCount > 0, child: const Icon(Icons.notifications_outlined)),
            selectedIcon: Badge(label: _pendingCount > 0 ? Text('$_pendingCount') : null, isLabelVisible: _pendingCount > 0, child: const Icon(Icons.notifications)),
            label: '消息',
          ),
          const NavigationDestination(icon: Icon(Icons.person_outline), selectedIcon: Icon(Icons.person), label: '我的'),
        ],
      ),
    );
  }
}
