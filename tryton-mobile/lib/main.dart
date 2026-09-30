import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'rpc_client.dart';
import 'screens/login_screen.dart';
import 'screens/menu_screen.dart';

void main() {
  runApp(const TalonApp());
}

class TalonApp extends StatelessWidget {
  const TalonApp({super.key});

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider(
      create: (_) => SessionState(),
      child: MaterialApp(
        title: 'Talon',
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF1A5276)),
          useMaterial3: true,
        ),
        home: const _Home(),
      ),
    );
  }
}

class SessionState extends ChangeNotifier {
  SessionState() {
    client = RpcClient(baseUrl: const String.fromEnvironment('TALON_URL', defaultValue: 'http://10.0.2.2:8000'));
  }

  late final RpcClient client;
  bool loggedIn = false;

  Future<void> login(String database, String login, String password) async {
    client.database = database;
    await client.login(login, password);
    loggedIn = true;
    notifyListeners();
  }

  Future<void> logout() async {
    await client.logout();
    loggedIn = false;
    notifyListeners();
  }
}

class _Home extends StatelessWidget {
  const _Home();

  @override
  Widget build(BuildContext context) {
    final session = context.watch<SessionState>();
    if (!session.loggedIn) {
      return const LoginScreen();
    }
    return const MenuScreen();
  }
}
