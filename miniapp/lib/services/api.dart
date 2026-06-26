import 'dart:convert';
import 'package:http/http.dart' as http;
import '../config/api.dart';

/// 统一 API 调用
class ApiService {
  static Future<Map<String, dynamic>> get(String path, {Map<String, String>? params}) async {
    final token = await ApiConfig.getToken();
    final merged = <String, String>{...?params, ...ApiConfig.viewAsParam};
    final uri = Uri.parse('${ApiConfig.baseUrl}$path').replace(queryParameters: merged.isNotEmpty ? merged : null);
    final res = await http.get(uri, headers: _headers(token));
    return _handle(res);
  }

  static Future<Map<String, dynamic>> post(String path, Map<String, dynamic> body) async {
    final token = await ApiConfig.getToken();
    final uri = Uri.parse('${ApiConfig.baseUrl}$path');
    final res = await http.post(uri, headers: _headers(token), body: jsonEncode(body));
    return _handle(res);
  }

  static Future<Map<String, dynamic>> put(String path, Map<String, dynamic> body) async {
    final token = await ApiConfig.getToken();
    final uri = Uri.parse('${ApiConfig.baseUrl}$path');
    final res = await http.put(uri, headers: _headers(token), body: jsonEncode(body));
    return _handle(res);
  }

  static Map<String, String> _headers(String? token) => {
    'Content-Type': 'application/json',
    if (token != null) 'Authorization': 'Bearer $token',
  };

  static Map<String, dynamic> _handle(http.Response res) {
    final data = jsonDecode(res.body);
    if (data['code'] != 0) throw ApiException(data['message'] ?? '请求失败');
    return data['data'] ?? {};
  }
}

class ApiException implements Exception {
  final String message;
  ApiException(this.message);
  @override
  String toString() => message;
}
