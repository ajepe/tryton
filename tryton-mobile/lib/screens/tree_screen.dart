import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../main.dart';
import 'form_screen.dart';

class TreeScreen extends StatefulWidget {
  const TreeScreen({super.key, required this.model});

  final String model;

  @override
  State<TreeScreen> createState() => _TreeScreenState();
}

class _TreeScreenState extends State<TreeScreen> {
  List<Map<String, dynamic>> _records = const [];
  Map<String, dynamic> _fields = const {};
  List<String> _columns = const [];
  String _search = '';
  bool _loading = true;

  static const _preferred = ['name', 'code', 'number', 'state', 'date', 'amount', 'active'];

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    final client = context.read<SessionState>().client;
    try {
      final fields = await client.fieldsGet(widget.model);
      final available = fields.keys.toList();
      var columns = _preferred.where(available.contains).take(5).toList();
      if (columns.isEmpty) {
        columns = available.where((name) {
          final type = (fields[name] as Map)['type'];
          return type != 'one2many' && type != 'many2many';
        }).take(4).toList();
      }
      final domain = _search.isEmpty
          ? const []
          : <dynamic>[
              [available.contains('name') ? 'name' : 'code', 'ilike', _search]
            ];
      final records = await client.searchRead(widget.model, domain, columns);
      setState(() {
        _fields = fields;
        _columns = columns;
        _records = records;
        _loading = false;
      });
    } catch (error) {
      setState(() => _loading = false);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.toString())));
      }
    }
  }

  String _format(dynamic value) {
    if (value == null) return '';
    if (value is bool) return value ? '✓' : '';
    if (value is DateTime) return value.toIso8601String().split('T').first;
    return value.toString();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.model),
        actions: [
          IconButton(icon: const Icon(Icons.add), onPressed: () => _openForm(0)),
          IconButton(icon: const Icon(Icons.refresh), onPressed: _load),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(8),
            child: TextField(
              decoration: const InputDecoration(
                hintText: 'Search…',
                prefixIcon: Icon(Icons.search),
                isDense: true,
              ),
              onSubmitted: (value) {
                _search = value;
                _load();
              },
            ),
          ),
          Expanded(
            child: _loading
                ? const Center(child: CircularProgressIndicator())
                : RefreshIndicator(
                    onRefresh: _load,
                    child: _records.isEmpty
                        ? const Center(child: Text('No records.'))
                        : ListView.separated(
                            itemCount: _records.length,
                            separatorBuilder: (_, __) => const Divider(height: 1),
                            itemBuilder: (context, index) {
                              final record = _records[index];
                              return ListTile(
                                title: Text(
                                  record[_columns.isNotEmpty ? _columns.first : 'id'].toString(),
                                ),
                                subtitle: Text(_columns
                                    .skip(1)
                                    .map((column) => _format(record[column]))
                                    .where((value) => value.isNotEmpty)
                                    .join(' · ')),
                                onTap: () => _openForm(record['id'] as int),
                              );
                            },
                          ),
                  ),
          ),
        ],
      ),
    );
  }

  Future<void> _openForm(int id) async {
    await Navigator.of(context).push(
      MaterialPageRoute(builder: (_) => FormScreen(model: widget.model, id: id)),
    );
    _load();
  }
}
