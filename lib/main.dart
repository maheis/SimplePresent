import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:path_provider/path_provider.dart';

void main() => runApp(const SimplePresentApp());

class SimplePresentApp extends StatelessWidget {
  const SimplePresentApp({super.key});
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'SimplePresent',
      theme: ThemeData(primarySwatch: Colors.blue),
      home: const HomePage(),
    );
  }
}

class HomePage extends StatefulWidget {
  const HomePage({super.key});
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  final List<String> _today = [];
  final TextEditingController _controller = TextEditingController();

  late final Future<void> _initFuture = _loadToday();

  Future<Directory> get _appDir async {
    final dir = await getApplicationDocumentsDirectory();
    return dir;
  }

  Future<File> _fileFor(String name) async {
    final dir = await _appDir;
    return File('${dir.path}/$name');
  }

  Future<void> _loadList(String filename, List<String> target) async {
    try {
      final f = await _fileFor(filename);
      if (await f.exists()) {
        final text = await f.readAsString();
        final data = jsonDecode(text) as List<dynamic>;
        target.clear();
        target.addAll(data.cast<String>());
      }
    } catch (_) {}
  }

  Future<void> _saveList(String filename, List<String> source) async {
    try {
      final f = await _fileFor(filename);
      await f.writeAsString(jsonEncode(source));
    } catch (_) {}
  }

  Future<void> _loadToday() async {
    await _loadList('tasks_today.json', _today);
    setState(() {});
  }

  Future<void> _saveToday() async {
    await _saveList('tasks_today.json', _today);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _addToToday(String text) {
    if (text.trim().isEmpty) return;
    setState(() {
      _today.insert(0, text.trim());
      _controller.clear();
    });
    _saveToday();
  }

  void _removeFromToday(int index) {
    setState(() {
      _today.removeAt(index);
    });
    _saveToday();
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<void>(
      future: _initFuture,
      builder: (context, snap) {
        return Scaffold(
          appBar: AppBar(title: const Text('SimplePresent')),
          body: Column(
            children: [
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('Heute', style: TextStyle(fontSize: 20, fontWeight: FontWeight.w600)),
                      const SizedBox(height: 8),
                      Expanded(
                        child: _today.isEmpty
                            ? const Center(child: Text('Keine Aufgaben für heute'))
                            : ListView.builder(
                                itemCount: _today.length,
                                itemBuilder: (ctx, i) => Dismissible(
                                  key: ValueKey('today_${i}_${_today[i]}'),
                                  background: Container(
                                    color: Colors.red,
                                    alignment: Alignment.centerRight,
                                    padding: const EdgeInsets.only(right: 20),
                                    child: const Icon(Icons.delete, color: Colors.white),
                                  ),
                                  direction: DismissDirection.endToStart,
                                  onDismissed: (_) => _removeFromToday(i),
                                  child: Card(child: ListTile(title: Text(_today[i]))),
                                ),
                              ),
                      ),
                    ],
                  ),
                ),
              ),
              SafeArea(
                top: false,
                child: Padding(
                  padding: const EdgeInsets.all(8.0),
                  child: Row(
                    children: [
                      Expanded(
                        child: TextField(
                          controller: _controller,
                          textInputAction: TextInputAction.done,
                          decoration: const InputDecoration(
                            hintText: 'Neue Aufgabe für heute',
                            border: OutlineInputBorder(),
                          ),
                          onSubmitted: _addToToday,
                        ),
                      ),
                      const SizedBox(width: 8),
                      ElevatedButton(
                        onPressed: () => _addToToday(_controller.text),
                        child: const Icon(Icons.add),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}
