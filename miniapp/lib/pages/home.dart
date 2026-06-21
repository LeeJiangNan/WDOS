import 'package:flutter/material.dart';
import '../services/api.dart';

class HomePage extends StatefulWidget {
  const HomePage({super.key});
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int _pending = 0, _processing = 0, _done = 0, _overtime = 0;
  List<Map<String, dynamic>> _alarms = [];

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final data = await ApiService.get('/stats/my-overview');
      setState(() {
        _pending = data['today']?['pending_count'] ?? 0;
        _processing = data['today']?['processing_count'] ?? 0;
        _done = data['today']?['completed_count'] ?? 0;
        _overtime = data['today']?['overtime_count'] ?? 0;
      });
    } catch (_) {}
    // 加载最新报警
    try {
      final res = await ApiService.get('/work-orders/pending');
      setState(() { _alarms = List<Map<String, dynamic>>.from(res['list'] ?? []); });
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('WDOS 工作台')),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(padding: const EdgeInsets.all(16), children: [
          // 统计卡片
          Row(children: [
            _statCard('待接单', _pending, Colors.red),
            const SizedBox(width: 12),
            _statCard('待处理', _processing, Colors.orange),
          ]),
          const SizedBox(height: 12),
          Row(children: [
            _statCard('今日完成', _done, Colors.green),
            const SizedBox(width: 12),
            _statCard('已超时', _overtime, Colors.red, blink: _overtime > 0),
          ]),
          const SizedBox(height: 20),
          Text('── 最新报警 ──', style: Theme.of(context).textTheme.titleSmall),
          ..._alarms.map((a) => Card(
            child: ListTile(
              leading: CircleAvatar(child: Text('${a['degree'] ?? '?'}')),
              title: Text(a['title'] ?? ''),
              subtitle: Text(a['camera_name'] ?? ''),
              trailing: Text(a['created_at']?.toString().substring(11, 19) ?? ''),
            ),
          )),
        ]),
      ),
    );
  }

  Widget _statCard(String label, int count, Color color, {bool blink = false}) {
    return Expanded(
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(children: [
            Text('$count', style: TextStyle(fontSize: 28, fontWeight: FontWeight.bold, color: color)),
            Text(label, style: const TextStyle(color: Colors.grey)),
          ]),
        ),
      ),
    );
  }
}
