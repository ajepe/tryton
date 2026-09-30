import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../main.dart';

class FormScreen extends StatefulWidget {
  const FormScreen({super.key, required this.model, required this.id});

  final String model;
  final int id;

  @override
  State<FormScreen> createState() => _FormScreenState();
}

class _FormScreenState extends State<FormScreen> {
  Map<String, dynamic> _fields = const {};
  Map<String, dynamic> _values = {};
  List<String> _editable = const [];
  bool _loading = true;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<SessionState>().client;
    try {
      final fields = await client.fieldsGet(widget.model);
      final editable = fields.keys.where((name) {
        final field = (fields[name] as Map).cast<String, dynamic>();
        return field['type'] != 'one2many' &&
            field['type'] != 'many2many' &&
            field['readonly'] != true;
      }).toList();
      Map<String, dynamic> values = {};
      if (widget.id > 0) {
        final records = await client.read(widget.model, [widget.id], fields.keys.toList());
        values = records.first;
      }
      setState(() {
        _fields = fields;
        _editable = editable;
        _values = values;
        _loading = false;
      });
    } catch (error) {
      setState(() => _loading = false);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.toString())));
      }
    }
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    final client = context.read<SessionState>().client;
    try {
      if (widget.id > 0) {
        await client.write(widget.model, [widget.id], _values);
      } else {
        await client.create(widget.model, _values);
      }
      if (mounted) Navigator.of(context).pop();
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(error.toString())));
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Widget _input(String name, Map<String, dynamic> field) {
    final type = field['type'] as String? ?? 'char';
    final value = _values[name];
    if (type == 'boolean') {
      return SwitchListTile(
        title: Text(field['string']?.toString() ?? name),
        value: value == true,
        onChanged: (checked) => setState(() => _values[name] = checked),
      );
    }
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: TextFormField(
        initialValue: value?.toString() ?? '',
        decoration: InputDecoration(
          labelText: field['string']?.toString() ?? name,
          border: const OutlineInputBorder(),
          isDense: true,
        ),
        keyboardType: (type == 'integer' || type == 'float' || type == 'numeric')
            ? TextInputType.number
            : TextInputType.text,
        onChanged: (text) => _values[name] = text,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('${widget.model} ${widget.id > 0 ? '#${widget.id}' : '(new)'}'),
        actions: [
          IconButton(
            icon: const Icon(Icons.check),
            onPressed: _saving ? null : _save,
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(12),
              children: _editable
                  .map((name) => _input(name, (_fields[name] as Map).cast<String, dynamic>()))
                  .toList(),
            ),
    );
  }
}
