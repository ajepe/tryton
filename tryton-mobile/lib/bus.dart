import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

class BusMessage {
  BusMessage(this.channel, this.data);

  final String channel;
  final Map<String, dynamic> data;
}

class BusClient {
  BusClient({
    required this.baseUrl,
    required this.database,
    required this.channels,
    required this.onMessage,
    required this.cookie,
  });

  final String baseUrl;
  final String database;
  final List<String> channels;
  final void Function(BusMessage message) onMessage;
  final String cookie;

  String? lastMessage;
  bool _running = false;

  Future<void> start() async {
    _running = true;
    while (_running) {
      try {
        await _poll();
      } catch (_) {
        await Future<void>.delayed(const Duration(seconds: 3));
      }
    }
  }

  void stop() {
    _running = false;
  }

  Future<void> _poll() async {
    final response = await http.post(
      Uri.parse('$baseUrl/${Uri.encodeComponent(database)}/bus'),
      headers: {'Content-Type': 'application/json', 'Cookie': cookie},
      body: jsonEncode({
        'channels': channels,
        'last_message': lastMessage,
      }),
    );
    if (response.statusCode != 200) {
      throw Exception('bus error ${response.statusCode}');
    }
    final messages = jsonDecode(response.body) as List<dynamic>;
    for (final item in messages) {
      final map = item as Map<String, dynamic>;
      final message = map['message'] as Map<String, dynamic>?;
      if (message == null) continue;
      lastMessage = message['id'] as String? ?? lastMessage;
      onMessage(BusMessage(
        map['channel'] as String? ?? '',
        (message['data'] as Map?)?.cast<String, dynamic>() ?? message,
      ));
    }
  }
}
