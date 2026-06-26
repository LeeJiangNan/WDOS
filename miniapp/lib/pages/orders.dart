import 'package:flutter/material.dart';
import '../services/api.dart';
import '../config/api.dart';
import '../widgets/order_detail.dart';

class OrdersPage extends StatefulWidget {
  const OrdersPage({super.key});
  @override
  State<OrdersPage> createState() => OrdersPageState();
}

class OrdersPageState extends State<OrdersPage> with SingleTickerProviderStateMixin {
  late TabController _tabCtrl;
  List<Map<String, dynamic>> _list = [];
  List<Map<String, dynamic>> _users = [];
  bool _loading = false;
  String _areaFilter = '';
  String _areaPattern = '';
  String _algoFilter = '';
  List<Map<String,String>> _areas = [];
  List<String> _algos = [];

  @override
  void initState() {
    super.initState();
    _tabCtrl = TabController(length: 3, vsync: this);
    _tabCtrl.addListener(() { if (!_tabCtrl.indexIsChanging) _fetch(); });
    _fetch();
    _loadUsers();

  }

  Future<void> _loadUsers() async {
    try {
      final data = await ApiService.get('/users');
      setState(() { _users = List<Map<String, dynamic>>.from(data['list'] ?? []); });
    } catch (_) {}
  }

  void _loadFilters() {
    ApiService.get('/area-routing-rules').then((res) {
      final rules = List<Map<String, dynamic>>.from(res['list'] ?? []);
      setState(() {
        _areas = rules.where((r) => r['is_active'] == true && (r['camera_group_pattern']?.toString() ?? '') != '*')
          .map((r) => {'pattern': r['camera_group_pattern']?.toString() ?? '', 'name': r['area_name']?.toString() ?? ''})
          .where((m) => m['pattern']!.isNotEmpty && m['name']!.isNotEmpty)
          .toList();
      });
    }).catchError((_) {});
    final algoSet = <String>{};
    for (var o in _list) {
      final algo = o['algorithm_name']?.toString() ?? '';
      if (algo.isNotEmpty) algoSet.add(algo);
    }
    _algos = algoSet.toList()..sort();
  }

  Future<void> _fetch() async {
    setState(() => _loading = true);
    final ep = ['/work-orders/pending', '/work-orders/processing', '/work-orders/completed'][_tabCtrl.index];
    try {
      final data = await ApiService.get(ep);
      setState(() { _list = List<Map<String, dynamic>>.from(data['list'] ?? []); });
    } catch (_) {}
    setState(() => _loading = false);
    _loadFilters();
  }

  void switchToTab(int index) { _tabCtrl.animateTo(index); _fetch(); }

  Future<void> _quickAccept(dynamic id) async {
    try { await ApiService.post('/work-orders/$id/accept', {}); _fetch(); }
    on ApiException catch (e) { if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.message))); }
  }

  Color _degreeColor(dynamic d) => [Colors.grey, Colors.blue, Colors.orange, Colors.red, Colors.deepPurple][(d is int ? d : 0).clamp(0, 4)];
  String _degreeLabel(dynamic d) => ['0', 'Ⅰ', 'Ⅱ', 'Ⅲ', 'Ⅳ'][(d is int ? d : 0).clamp(0, 4)];

  String _timeAgo(String? t) {
    if (t == null) return '';
    try {
      final dt = DateTime.parse(t);
      final diff = DateTime.now().difference(dt);
      if (diff.inSeconds < 60) return '${diff.inSeconds}秒前';
      if (diff.inMinutes < 60) return '${diff.inMinutes}分钟前';
      if (diff.inHours < 24) return '${diff.inHours}小时前';
      return '${diff.inDays}天前';
    } catch (_) { return t.length > 16 ? t.substring(5, 16) : t; }
  }

  Widget _statusBadge(String s) {
    final c = {'pending': ('待接单', Colors.red), 'processing': ('处理中', Colors.orange), 'completed': ('已完成', Colors.green)}[s] ?? (s, Colors.grey);
    final (label, color) = c;
    return Container(padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 2), decoration: BoxDecoration(color: color.withOpacity(0.1), borderRadius: BorderRadius.circular(4)), child: Text(label, style: TextStyle(fontSize: 10, color: color)));
  }

  Widget _noImg(dynamic d) => Container(width: 56, height: 42, color: _degreeColor(d).withOpacity(0.1), child: Icon(Icons.image, size: 22, color: _degreeColor(d)));

  List<Map<String, dynamic>> get _filtered {
    var list = _list;
    if (_areaPattern.isNotEmpty) { final pfx = _areaPattern.replaceAll('*', ''); list = list.where((a) => (a['camera_name']?.toString() ?? '').startsWith(pfx)).toList(); }
    if (_algoFilter.isNotEmpty) list = list.where((a) => (a['algorithm_name']?.toString() ?? '').contains(_algoFilter)).toList();
    return list;
  }

  void _showFilter() {
    showModalBottomSheet(
      context: context,
      builder: (ctx) => StatefulBuilder(builder: (ctx, setSt) => Padding(
        padding: const EdgeInsets.all(20),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Text('筛选工单', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
          const SizedBox(height: 16),
          if (_areas.isNotEmpty) ...[
            const Text('按区域', style: TextStyle(fontSize: 14, color: Colors.grey)),
            const SizedBox(height: 8),
            Wrap(spacing: 8, runSpacing: 6, children: [
              FilterChip(label: const Text('全部'), selected: _areaFilter.isEmpty, onSelected: (_) { setSt(() => _areaFilter = ''); Navigator.pop(ctx); setState(() {}); }),
              ..._areas.map((a) => FilterChip(label: Text(a["name"]!), selected: _areaFilter == a["name"], onSelected: (_) { setSt(() { _areaFilter = a["name"]!; _areaPattern = a["pattern"]!; }); Navigator.pop(ctx); setState(() {}); })),
            ]),
            const SizedBox(height: 20),
          ],
          if (_algos.isNotEmpty) ...[
            const Text('按算法', style: TextStyle(fontSize: 14, color: Colors.grey)),
            const SizedBox(height: 8),
            Wrap(spacing: 8, runSpacing: 6, children: [
              FilterChip(label: const Text('全部'), selected: _algoFilter.isEmpty, onSelected: (_) { setSt(() => _algoFilter = ''); Navigator.pop(ctx); setState(() {}); }),
              ..._algos.map((a) => FilterChip(label: Text(a), selected: _algoFilter == a, onSelected: (_) { setSt(() => _algoFilter = a); Navigator.pop(ctx); setState(() {}); })),
            ]),
          ],
          const SizedBox(height: 10),
        ]),
      )),
    );
  }

  void _openDetail(dynamic o) => showOrderDetail(context, order: o, users: _users, onChanged: _fetch);

  @override
  Widget build(BuildContext context) {
    final filtered = _filtered;
    return Scaffold(
      appBar: AppBar(
        title: const Text('工单中心'),
        actions: [IconButton(icon: const Icon(Icons.filter_list), onPressed: _showFilter, tooltip: '筛选')],
        bottom: TabBar(controller: _tabCtrl, tabs: const [Tab(text: '待接单'), Tab(text: '待处理'), Tab(text: '已完成')]),
      ),
      body: _loading ? const Center(child: CircularProgressIndicator())
        : RefreshIndicator(
            onRefresh: _fetch,
            child: Column(children: [
              if (_areaFilter.isNotEmpty || _algoFilter.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.fromLTRB(10, 8, 10, 0),
                  child: Row(children: [
                    Text('筛选: ', style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
                    if (_areaFilter.isNotEmpty) Chip(label: Text(_areaFilter, style: const TextStyle(fontSize: 12)), onDeleted: () => setState(() { _areaFilter = ''; _areaPattern = ''; }), materialTapTargetSize: MaterialTapTargetSize.shrinkWrap, visualDensity: VisualDensity.compact),
                    if (_algoFilter.isNotEmpty) Chip(label: Text(_algoFilter, style: const TextStyle(fontSize: 12)), onDeleted: () => setState(() => _algoFilter = ''), materialTapTargetSize: MaterialTapTargetSize.shrinkWrap, visualDensity: VisualDensity.compact),
                  ]),
                ),
              Expanded(
                child: filtered.isEmpty
                  ? ListView(children: const [SizedBox(height: 120), Center(child: Text('暂无工单', style: TextStyle(color: Colors.grey)))])
                  : ListView.builder(
                      itemCount: filtered.length,
                      itemBuilder: (ctx, i) {
                        final o = filtered[i];
                        final picUrl = o['alarm_pic_url']?.toString() ?? '';
                        final dup = o['duplicate_count'] ?? 1;
                        return Card(
                          margin: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                          child: InkWell(
                            onTap: () => _openDetail(o),
                            borderRadius: BorderRadius.circular(12),
                            child: Padding(
                              padding: const EdgeInsets.all(10),
                              child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                                ClipRRect(borderRadius: BorderRadius.circular(6),
                                  child: picUrl.isNotEmpty
                                      ? Image.network(ApiConfig.imageUrl(picUrl), width: 56, height: 42, fit: BoxFit.cover, errorBuilder: (_, __, ___) => _noImg(o['degree']))
                                      : _noImg(o['degree'])),
                                const SizedBox(width: 8),
                                Expanded(
                                  child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                                    Row(children: [
                                      Container(padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1), decoration: BoxDecoration(color: _degreeColor(o['degree']).withOpacity(0.15), borderRadius: BorderRadius.circular(3)), child: Text(_degreeLabel(o['degree']), style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: _degreeColor(o['degree'])))),
                                      const SizedBox(width: 4),
                                      if (dup > 1) Text('×$dup', style: TextStyle(fontSize: 10, color: Colors.orange.shade700)),
                                      const SizedBox(width: 4),
                                      _statusBadge(o['status']?.toString() ?? ''),
                                    ]),
                                    const SizedBox(height: 3),
                                    Text(o['title'] ?? '', style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14), maxLines: 2, overflow: TextOverflow.ellipsis),
                                    Text('📍 ${o['camera_name'] ?? ''} · ${_timeAgo(o['created_at']?.toString())}', style: const TextStyle(fontSize: 11, color: Colors.grey)),
                                  ]),
                                ),
                                if (o['status'] == 'pending')
                                  Padding(padding: const EdgeInsets.only(left: 4), child: FilledButton(onPressed: () => _quickAccept(o['id']), style: FilledButton.styleFrom(padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4), minimumSize: Size.zero), child: const Text('接单', style: TextStyle(fontSize: 13)))),
                              ]),
                            ),
                          ),
                        );
                      },
                    ),
              ),
            ]),
          ),
    );
  }
}
