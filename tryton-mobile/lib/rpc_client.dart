import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import 'codec.dart';

class RpcException implements Exception {
  final String type;
  final List<dynamic> args;

  RpcException(this.type, this.args);

  @override
  String toString() => args.isNotEmpty ? args.first.toString() : type;
}

List<List<String>> _orderParams(String order) {
  if (order.isEmpty) return const [];
  return order
      .split(',')
      .map((item) => item.trim())
      .where((item) => item.isNotEmpty)
      .map((item) {
    final parts = item.split(RegExp(r'\s+'));
    return [parts.first, (parts.length > 1 ? parts[1] : 'ASC').toUpperCase()];
  }).toList();
}

class RpcClient {
  RpcClient({required this.baseUrl, this.database = ''});

  final String baseUrl;
  String database;
  String? sessionCookie;
  int userId = 0;

  String get _prefix => '$baseUrl/${Uri.encodeComponent(database)}';

  Future<Map<String, String>> _headers() async {
    final headers = {'Content-Type': 'application/json'};
    if (sessionCookie != null) {
      headers['Cookie'] = sessionCookie!;
    }
    return headers;
  }

  Future<dynamic> call(String method, [List<dynamic> params = const []]) async {
    final response = await http.post(
      Uri.parse('$_prefix/rpc/'),
      headers: await _headers(),
      body: jsonEncode({
        'id': 1,
        'method': method,
        'params': encodeValue(params),
      }),
    );
    final payload = jsonDecode(response.body) as Map<String, dynamic>;
    if (payload['error'] != null) {
      final error = payload['error'] as List<dynamic>;
      throw RpcException(error[0] as String, (error[1] as List).cast<dynamic>());
    }
    return decodeValue(payload['result']);
  }

  Future<List<String>> listDatabases() async {
    final result = await call('common.db.list');
    return (result as List).map((item) => item.toString()).toList();
  }

  Future<void> login(String login, String password) async {
    final response = await http.post(
      Uri.parse('$_prefix/session/login'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'id': 0,
        'method': 'common.db.login',
        'params': [login, {'password': password}, 'en'],
      }),
    );
    final payload = jsonDecode(response.body) as Map<String, dynamic>;
    if (payload['error'] != null) {
      final error = payload['error'] as List<dynamic>;
      throw RpcException(error[0] as String, (error[1] as List).cast<dynamic>());
    }
    final cookie = response.headers['set-cookie'];
    if (cookie != null) {
      sessionCookie = cookie.split(';').first;
      final preferences = await SharedPreferences.getInstance();
      await preferences.setString('talon_session', sessionCookie!);
    }
    final result = decodeValue(payload['result']) as List<dynamic>;
    userId = result[0] as int;
  }

  Future<void> restoreSession() async {
    final preferences = await SharedPreferences.getInstance();
    sessionCookie = preferences.getString('talon_session');
  }

  Future<void> logout() async {
    await http.post(
      Uri.parse('$_prefix/session/logout'),
      headers: await _headers(),
      body: jsonEncode({'id': 0, 'method': 'common.db.logout', 'params': []}),
    );
    sessionCookie = null;
    final preferences = await SharedPreferences.getInstance();
    await preferences.remove('talon_session');
  }

  Future<List<Map<String, dynamic>>> searchRead(
    String model,
    List<dynamic> domain,
    List<String> fields, {
    int offset = 0,
    int limit = 40,
    String order = '',
  }) async {
    final result = await call('model.$model.search_read',
        [domain, offset, limit, _orderParams(order), fields, const {}]);
    return (result as List).cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> fieldsGet(String model) async {
    final result = await call('model.$model.fields_get', [const [], const {'language': 'en'}]);
    return (result as Map).cast<String, dynamic>();
  }

  Future<List<Map<String, dynamic>>> read(String model, List<int> ids, List<String> fields) async {
    final result = await call('model.$model.read', [ids, fields, const {}]);
    return (result as List).cast<Map<String, dynamic>>();
  }

  Future<List<int>> create(String model, Map<String, dynamic> values) async {
    final result = await call('model.$model.create', [
      [values],
      const {},
    ]);
    return (result as List).map((id) => id as int).toList();
  }

  Future<void> write(String model, List<int> ids, Map<String, dynamic> values) async {
    await call('model.$model.write', [ids, values, const {}]);
  }

  Future<void> remove(String model, List<int> ids) async {
    await call('model.$model.delete', [ids, const {}]);
  }

  Future<Map<String, dynamic>> onChanged(
      String model, Map<String, dynamic> values, List<String> fields) async {
    final result = await call('model.$model.on_change', [values, fields, const {}]);
    return (result as Map).cast<String, dynamic>();
  }
}
