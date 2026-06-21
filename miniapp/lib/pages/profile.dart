import 'package:flutter/material.dart';
import '../services/api.dart';
import '../config/api.dart';

class ProfilePage extends StatefulWidget {
  const ProfilePage({super.key});
  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  int _done = 0, _rank = 0, _total = 0;
  double _avgSeconds = 0;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final data = await ApiService.get('/stats/my-overview');
      setState(() {
        _done = data['today']?['completed_count'] ?? 0;
        _avgSeconds = (data['week']?['avg_process_seconds'] ?? 0).toDouble();
        _rank = data['week']?['rank'] ?? 0;
        _total = data['week']?['completed_count'] ?? 0;
      });
    } catch (_) {}
  }

  Future<void> _logout() async {
    await ApiConfig.clearToken();
    if (mounted) Navigator.pushReplacement(context, MaterialPageRoute(builder: (_) => const SizedBox()));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('我的')),
      body: ListView(padding: const EdgeInsets.all(16), children: [
        const Card(child: Padding(padding: EdgeInsets.all(16), child: Row(children: [
          CircleAvatar(radius: 28, child: Icon(Icons.person, size: 32)),
          SizedBox(width: 16),
          Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text('张三', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
            Text('安保部 | 一线保安', style: TextStyle(color: Colors.grey)),
          ]),
        ]))),
        const SizedBox(height: 16),
        Row(children: [
          _statCol('完成工单', '$_done', Colors.green),
          _statCol('平均耗时', '${_avgSeconds.toInt()}s', Colors.blue),
          _statCol('今日排名', '$_rank/$_total', Colors.orange),
        ]),
        const SizedBox(height: 16),
        Card(child: Column(children: [
          ListTile(leading: const Icon(Icons.assignment), title: const Text('我的工单'), trailing: const Icon(Icons.chevron_right), onTap: () {}),
          ListTile(leading: const Icon(Icons.bar_chart), title: const Text('数据统计'), trailing: const Icon(Icons.chevron_right), onTap: () {}),
          ListTile(leading: const Icon(Icons.calendar_today), title: const Text('我的排班'), trailing: const Icon(Icons.chevron_right), onTap: () {}),
        ])),
        const SizedBox(height: 16),
        FilledButton.tonal(onPressed: _logout, child: const Text('退出登录')),
      ]),
    );
  }

  Widget _statCol(String label, String value, Color color) {
    return Expanded(
      child: Card(child: Padding(padding: const EdgeInsets.all(12), child: Column(children: [
        Text(value, style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold, color: color)),
        Text(label, style: const TextStyle(fontSize: 12, color: Colors.grey)),
      ]))),
    );
  }
}
