import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../bus.dart';
import '../main.dart';
import 'tree_screen.dart';

class MenuScreen extends StatefulWidget {
  const MenuScreen({super.key});

  @override
  State<MenuScreen> createState() => _MenuScreenState();
}

class _MenuScreenState extends State<MenuScreen> {
  List<Map<String, dynamic>> _menus = const [];
  List<Map<String, dynamic>> _models = const [];
  String _filter = '';
  int _notificationCount = 0;
  BusClient? _bus;

  @override
  void initState() {
    super.initState();
    _load();
    _startBus();
  }

  @override
  void dispose() {
    _bus?.stop();
    super.dispose();
  }

  Future<void> _load() async {
    final client = context.read<SessionState>().client;
    try {
      final menus = await client.searchRead(
          'ir.ui.menu', const [], const ['name', 'action', 'parent'],
          limit: 200);
      final models =
          await client.searchRead('ir.model', const [], const ['name', 'model'], limit: 200);
      setState(() {
        _menus = menus;
        _models = models;
      });
    } catch (_) {
      setState(() {
        _menus = const [];
        _models = const [];
      });
    }
  }

  void _startBus() {
    final session = context.read<SessionState>();
    if (session.client.sessionCookie == null) return;
    _bus = BusClient(
      baseUrl: session.client.baseUrl,
      database: session.client.database,
      channels: ['notification:${session.client.userId}'],
      cookie: session.client.sessionCookie!,
      onMessage: (_) => setState(() => _notificationCount++),
    );
    _bus!.start();
  }

  Future<void> _openModel(String model) async {
    await Navigator.of(context).push(
      MaterialPageRoute(builder: (_) => TreeScreen(model: model)),
    );
  }

  @override
  Widget build(BuildContext context) {
    final filteredModels = _models
        .where((model) =>
            '${model['name']} ${model['model']}'.toLowerCase().contains(_filter.toLowerCase()))
        .toList();
    return Scaffold(
      appBar: AppBar(
        title: const Text('Talon'),
        actions: [
          IconButton(
            icon: Badge(
              isLabelVisible: _notificationCount > 0,
              label: Text('$_notificationCount'),
              child: const Icon(Icons.notifications),
            ),
            onPressed: () => setState(() => _notificationCount = 0),
          ),
          IconButton(
            icon: const Icon(Icons.logout),
            onPressed: () => context.read<SessionState>().logout(),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          const Text('Menu', style: TextStyle(fontWeight: FontWeight.bold)),
          ..._menus.map((menu) => ListTile(
                title: Text(menu['name']?.toString() ?? ''),
                dense: true,
                onTap: () {
                  final action = menu['action'];
                  if (action is String && action.contains(',')) {
                    _openModel('');
                  }
                },
              )),
          const Divider(),
          const Text('Models', style: TextStyle(fontWeight: FontWeight.bold)),
          TextField(
            decoration: const InputDecoration(hintText: 'Filter models…', isDense: true),
            onChanged: (value) => setState(() => _filter = value),
          ),
          ...filteredModels.map((model) => ListTile(
                title: Text(model['name']?.toString() ?? ''),
                subtitle: Text(model['model']?.toString() ?? ''),
                dense: true,
                onTap: () => _openModel(model['model'].toString()),
              )),
          if (filteredModels.isEmpty && _filter.isNotEmpty)
            ListTile(
              title: Text('Open "$_filter"'),
              trailing: const Icon(Icons.arrow_forward),
              onTap: () => _openModel(_filter),
            ),
        ],
      ),
    );
  }
}
