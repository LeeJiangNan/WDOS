import 'package:shared_preferences/shared_preferences.dart';

/// API 配置
class ApiConfig {
  // 开发环境
  static const String baseUrl = 'http://localhost:8080/api/v1'; // Android 模拟器
  // static const String baseUrl = 'http://localhost:8080/api/v1'; // iOS 模拟器
  // static const String baseUrl = 'https://wdos.yourmall.com/api/v1'; // 生产

  static const String wsUrl = 'ws://10.0.2.2:8080/ws/notifications';

  static Future<String?> getToken() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString('wdos_token');
  }

  static Future<void> setToken(String token) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('wdos_token', token);
  }

  static Future<void> clearToken() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove('wdos_token');
    await prefs.remove('wdos_user');
  }
}
