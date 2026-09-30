import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../main.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _database = TextEditingController();
  final _login = TextEditingController(text: 'admin');
  final _password = TextEditingController();
  List<String> _databases = const [];
  String? _error;
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    _loadDatabases();
  }

  Future<void> _loadDatabases() async {
    try {
      final client = context.read<SessionState>().client;
      final databases = await client.listDatabases();
      setState(() {
        _databases = databases;
        if (databases.isNotEmpty && _database.text.isEmpty) {
          _database.text = databases.first;
        }
      });
    } catch (error) {
      setState(() => _error = error.toString());
    }
  }

  Future<void> _submit() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      await context.read<SessionState>().login(_database.text, _login.text, _password.text);
    } catch (error) {
      setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 380),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text('Talon', style: Theme.of(context).textTheme.headlineMedium),
                const SizedBox(height: 24),
                DropdownButtonFormField<String>(
                  initialValue: _databases.contains(_database.text) ? _database.text : null,
                  decoration: const InputDecoration(labelText: 'Database'),
                  items: _databases
                      .map((name) => DropdownMenuItem(value: name, child: Text(name)))
                      .toList(),
                  onChanged: (value) => setState(() => _database.text = value ?? ''),
                ),
                TextField(
                  controller: _login,
                  decoration: const InputDecoration(labelText: 'Login'),
                  autofillHints: const [AutofillHints.username],
                ),
                TextField(
                  controller: _password,
                  obscureText: true,
                  decoration: const InputDecoration(labelText: 'Password'),
                  autofillHints: const [AutofillHints.password],
                ),
                if (_error != null) ...[
                  const SizedBox(height: 12),
                  Text(_error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
                ],
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: _loading ? null : _submit,
                  child: Text(_loading ? 'Signing in…' : 'Sign in'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
