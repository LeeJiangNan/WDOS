import 'package:flutter/material.dart';
import '../services/api.dart';
import '../widgets/order_detail.dart';

class MessagesPage extends StatefulWidget {
  const MessagesPage({super.key});
  @override
  State<MessagesPage> createState() => _MessagesPageState();
}

class _MessagesPageState extends State<MessagesPage> {
  List<Map<String, dynamic>> _orders = [];
  List<Map<String, dynamic>> _users = [];
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    _fetch();
    _loadUsers();
  }

  Future<void> _loadUsers() async {
    try {
      final data = await ApiService.get('/users');
      setState(() { _users = List<Map<String, dynamic>>.from(data['list'] ?? []); });
    } catch (_) {}
  }

  Future<void> _fetch() async {
    setState(() => _loading = true);
    try {
      final data = await ApiService.get('/work-orders');
      setState(() {
        _orders = List<Map<String, dynamic>>.from(data['list'] ?? []);
      });
    } catch (_) {}
    setState(() => _loading = false);
  }

  IconData _iconForOrder(dynamic order) {
    final status = order['status'] ?? '';
    if (status == 'pending') return Icons.notifications_active;
    if (status == 'processing') return Icons.timer;
    if (status == 'completed') return Icons.check_circle;
    return Icons.info;
  }

  Color _colorForOrder(dynamic order) {
    final status = order['status'] ?? '';
    if (status == 'pending') return Colors.red;
    if (status == 'processing') return Colors.orange;
    if (status == 'completed') return Colors.green;
    return Colors.grey;
  }

  String _statusLabel(String s) {
    return {'pending': '待接单', 'processing': '处理中', 'completed': '已完成'}[s] ?? s;
  }

  // 点击工单 → 打开统一详情弹窗
  void _openDetail(dynamic order) {
    showOrderDetail(context, order: order, users: _users, onChanged: _fetch);
  }

  // 接单（带确认）
  Future<void> _quickAccept(dynamic order) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('确认接单'),
        content: Text('确定要接单「${order['title'] ?? ''}」吗？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('确认接单')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await ApiService.post('/work-orders/${order['id']}/accept', {});
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('已接单')));
      _fetch();
    } on ApiException catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.message)));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('消息中心'),
        actions: [IconButton(icon: const Icon(Icons.refresh), onPressed: _fetch)],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : RefreshIndicator(
              onRefresh: _fetch,
              child: _orders.isEmpty
                  ? ListView(children: const [
                      SizedBox(height: 120),
                      Center(child: Text('暂无消息', style: TextStyle(color: Colors.grey, fontSize: 16))),
                    ])
                  : ListView.builder(
                      itemCount: _orders.length,
                      itemBuilder: (ctx, i) {
                        final o = _orders[i];
                        return Card(
                          child: ListTile(
                            leading: Icon(_iconForOrder(o), color: _colorForOrder(o)),
                            title: Text(o['title'] ?? '', maxLines: 1, overflow: TextOverflow.ellipsis),
                            subtitle: Text('${_statusLabel(o['status'] ?? '')} · ${o['camera_name'] ?? ''}\n${o['created_at'] ?? ''}'),
                            trailing: o['status'] == 'pending'
                                ? TextButton(onPressed: () => _quickAccept(o), child: const Text('接单'))
                                : null,
                            onTap: () => _openDetail(o),
                            isThreeLine: true,
                          ),
                        );
                      },
                    ),
            ),
    );
  }
}
