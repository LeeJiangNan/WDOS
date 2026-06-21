import 'package:flutter/material.dart';
import '../services/api.dart';

class OrdersPage extends StatefulWidget {
  const OrdersPage({super.key});
  @override
  State<OrdersPage> createState() => _OrdersPageState();
}

class _OrdersPageState extends State<OrdersPage> with SingleTickerProviderStateMixin {
  late TabController _tabCtrl;
  List<Map<String, dynamic>> _list = [];
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    _tabCtrl = TabController(length: 3, vsync: this);
    _tabCtrl.addListener(() { if (!_tabCtrl.indexIsChanging) _fetch(); });
    _fetch();
  }

  Future<void> _fetch() async {
    setState(() => _loading = true);
    final ep = ['/work-orders/pending', '/work-orders/processing', '/work-orders/completed'][_tabCtrl.index];
    try {
      final data = await ApiService.get(ep);
      setState(() { _list = List<Map<String, dynamic>>.from(data['list'] ?? []); });
    } catch (_) {}
    setState(() => _loading = false);
  }

  Future<void> _accept(int id) async {
    try {
      await ApiService.post('/work-orders/$id/accept', {});
      _fetch();
    } on ApiException catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.message)));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('工单中心'),
        bottom: TabBar(controller: _tabCtrl, tabs: const [
          Tab(text: '待接单'), Tab(text: '待处理'), Tab(text: '已完成'),
        ]),
      ),
      body: _loading
        ? const Center(child: CircularProgressIndicator())
        : RefreshIndicator(
            onRefresh: _fetch,
            child: ListView.builder(
              itemCount: _list.length,
              itemBuilder: (ctx, i) {
                final o = _list[i];
                return Card(
                  child: ListTile(
                    title: Text(o['title'] ?? '', maxLines: 2, overflow: TextOverflow.ellipsis),
                    subtitle: Text('📍 ${o['camera_name'] ?? ''}\n'
                        '${o['status'] == 'pending' ? '⚠️ 待接单' : o['status'] == 'processing' ? '🔄 处理中' : '✅ 已完成'}'),
                    trailing: o['status'] == 'pending'
                        ? ElevatedButton(onPressed: () => _accept(o['id']), child: const Text('接单'))
                        : o['status'] == 'processing'
                            ? ElevatedButton(onPressed: () {/* 处理 */}, child: const Text('处理'))
                            : null,
                    onTap: () { /* 进入工单详情 */ },
                  ),
                );
              },
            ),
          ),
    );
  }
}
