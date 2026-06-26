import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'dart:convert';
import '../services/api.dart';
import '../config/api.dart';
import '../main.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({super.key});
  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _userCtrl = TextEditingController(text: 'admin');
  final _pwdCtrl = TextEditingController(text: 'Admin@123');
  bool _loading = false;

  Future<void> _login() async {
    setState(() => _loading = true);
    try {
      final data = await ApiService.post('/auth/login', {
        'username': _userCtrl.text,
        'password': _pwdCtrl.text,
      });
      await ApiConfig.setToken(data['access_token']);
      // 存储用户信息
      if (data['user'] != null) {
        final prefs = await SharedPreferences.getInstance();
        await prefs.setString('wdos_user', jsonEncode(data['user']));
      }
      if (mounted) Navigator.pushReplacement(context, MaterialPageRoute(builder: (_) => const MainTabs()));
    } on ApiException catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.message)));
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            const Text('🏬', style: TextStyle(fontSize: 64)),
            const SizedBox(height: 16),
            Text('WDOS 工单系统', style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 32),
            TextField(controller: _userCtrl, decoration: const InputDecoration(labelText: '用户名', border: OutlineInputBorder())),
            const SizedBox(height: 16),
            TextField(controller: _pwdCtrl, obscureText: true, decoration: const InputDecoration(labelText: '密码', border: OutlineInputBorder()), onSubmitted: (_) => _login()),
            const SizedBox(height: 24),
            SizedBox(width: double.infinity, height: 48,
              child: FilledButton(onPressed: _loading ? null : _login, child: _loading ? const CircularProgressIndicator() : const Text('登 录', style: TextStyle(fontSize: 16)))),
          ]),
        ),
      ),
    );
  }
}
