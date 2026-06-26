import 'package:shared_preferences/shared_preferences.dart';

/// API 配置
class ApiConfig {
  // 服务器地址 — 优先从当前页面 URL 推断（Web 端），否则用默认值
  static String _host = '100.107.124.26';

  static void init() {
    // Web 端：从浏览器地址栏获取主机名，自动适配 Tailscale/局域网
    try {
      // ignore: avoid_dynamic_calls
      final host = Uri.base.host;
      if (host.isNotEmpty) {
        _host = host;
      }
    } catch (_) {
      // 非 Web 平台，用默认值
    }
  }

  static String get baseUrl => 'http://$_host:9090/api/v1';
  static String get wsUrl => 'ws://$_host:9090/ws/notifications';

  // 领导切换身份查看下级数据
  static int? viewAsUserId;

  static Map<String, String> get viewAsParam {
    if (viewAsUserId != null) return {'view_as': viewAsUserId.toString()};
    return {};
  }

  // 拼接完整图片 URL：/minio/... → http://host:9090/api/v1/minio/...
  static String imageUrl(String path) {
    if (path.startsWith('http')) return path;
    return 'http://$_host:9090/api/v1$path';
  }

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
