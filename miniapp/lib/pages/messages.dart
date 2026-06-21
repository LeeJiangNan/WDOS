import 'package:flutter/material.dart';

class MessagesPage extends StatelessWidget {
  const MessagesPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('消息中心'), actions: [TextButton(onPressed: () {}, child: const Text('全部已读'))]),
      body: ListView(
        children: const [
          _MsgItem(icon: Icons.notifications_active, color: Colors.red, title: '新工单', desc: '行人闯入 - B1停车场C区3号通道\n请尽快接单处理', time: '15:04'),
          _MsgItem(icon: Icons.timer, color: Colors.orange, title: '接单超时提醒', desc: 'WD-1525 已超时30秒未接单\n已上报班长', time: '15:04'),
          _MsgItem(icon: Icons.check_circle, color: Colors.green, title: '工单完成', desc: 'WD-0890 车辆违停已处理完成', time: '昨天'),
        ],
      ),
    );
  }
}

class _MsgItem extends StatelessWidget {
  final IconData icon;
  final Color color;
  final String title, desc, time;
  const _MsgItem({required this.icon, required this.color, required this.title, required this.desc, required this.time});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: Icon(icon, color: color),
        title: Text(title, style: const TextStyle(fontWeight: FontWeight.w600)),
        subtitle: Text(desc, maxLines: 2),
        trailing: Text(time, style: const TextStyle(fontSize: 12, color: Colors.grey)),
      ),
    );
  }
}
