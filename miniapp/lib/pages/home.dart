import 'package:flutter/material.dart';
import '../services/api.dart';
import '../config/api.dart';
import '../widgets/order_detail.dart';

class HomePage extends StatefulWidget {
  final void Function(int tabIndex)? onNavigateOrders;
  const HomePage({super.key, this.onNavigateOrders});
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int _pending = 0, _processing = 0, _done = 0, _overtime = 0, _overtimePending = 0;
  List<Map<String, dynamic>> _alarms = [];
  List<Map<String, dynamic>> _users = [];
  String _areaFilter = '';
  String _areaPattern = '';
  String _algoFilter = '';
  List<Map<String,String>> _areas = [];
  List<String> _algos = [];
  String _range = 'today';
  // 统计缓存：切换范围时先用缓存，后台刷新
  final Map<String, Map<String, int>> _statsCache = {};

  @override
  void initState() {
    super.initState();
    _load();
    _loadFilters();
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
          .where((m) => m['pattern']!.isNotEmpty && m['name']!.isNotEmpty).toList();
      });
    }).catchError((_) {});
    final algoSet = <String>{};
    for (var o in _alarms) { final algo = o['algorithm_name']?.toString() ?? ''; if (algo.isNotEmpty) algoSet.add(algo); }
    _algos = algoSet.toList()..sort();
  }

  Future<void> _load() async {
    // 先用缓存立即更新UI，避免切换范围时卡顿
    final cached = _statsCache[_range];
    if (cached != null) {
      setState(() {
        _pending = cached['pending'] ?? 0;
        _processing = cached['processing'] ?? 0;
        _done = cached['done'] ?? 0;
        _overtime = cached['overtime'] ?? 0;
      });
    }
    // 后台刷新
    final ep = _range == 'week' ? '/stats/weekly-overview' : _range == 'month' ? '/stats/monthly-overview' : '/stats/my-overview';
    try {
      final data = await ApiService.get(ep);
      final p = data['pending_orders'] ?? 0;
      final r = data['processing_orders'] ?? 0;
      final d = data['completed_orders'] ?? 0;
      final o = data['overtime_orders'] ?? 0;
      _statsCache[_range] = {'pending': p, 'processing': r, 'done': d, 'overtime': o};
      if (mounted) setState(() { _pending = p; _processing = r; _done = d; _overtime = o; });
    } catch (_) {}
    // 工单列表首次加载或刷新时才拉
    if (_alarms.isEmpty || cached != null) {
      try {
        final res = await ApiService.get('/work-orders');
        final list = List<Map<String, dynamic>>.from(res['list'] ?? []);
        int otPend = 0;
        for (var o in list) {
          if (o['status'] == 'pending' && (o['escalated_level'] ?? 0) > 0) otPend++;
          if (o['status'] == 'processing' && (o['escalated_level'] ?? 0) > 0) otPend++;
        }
        if (mounted) setState(() { _alarms = list; _overtimePending = otPend; });
      } catch (_) {}
    }
    _loadFilters();
  }

  void _switchRange(String range) {
    if (_range == range) return;
    setState(() => _range = range);
    _load(); // 异步加载，缓存命中则UI秒切
  }

  List<Map<String, dynamic>> get _filteredAlarms {
    var list = _alarms;
    if (_areaPattern.isNotEmpty) { final pfx = _areaPattern.replaceAll('*', ''); list = list.where((a) => (a['camera_name']?.toString() ?? '').startsWith(pfx)).toList(); }
    if (_algoFilter.isNotEmpty) list = list.where((a) => (a['algorithm_name']?.toString() ?? '').contains(_algoFilter)).toList();
    return list;
  }

  Color _degreeColor(dynamic d) => [Colors.grey, Colors.blue, Colors.orange, Colors.red, Colors.deepPurple][(d is int ? d : 0).clamp(0, 4)];
  String _degreeLabel(dynamic d) => ['0', 'Ⅰ', 'Ⅱ', 'Ⅲ', 'Ⅳ'][(d is int ? d : 0).clamp(0, 4)];

  String _timeAgo(String? t) {
    if (t == null) return '';
    try {
      final dt = DateTime.parse(t); final diff = DateTime.now().difference(dt);
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

  Widget _noImg(dynamic d) => Container(width: 64, height: 48, color: _degreeColor(d).withOpacity(0.1), child: Icon(Icons.image, size: 22, color: _degreeColor(d)));

  void _showFilter() {
    showModalBottomSheet(context: context, builder: (ctx) => StatefulBuilder(builder: (ctx, setSt) => Padding(
      padding: const EdgeInsets.all(20), child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
        const Text('筛选工单', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)), const SizedBox(height: 16),
        if (_areas.isNotEmpty) ...[const Text('按区域', style: TextStyle(fontSize: 14, color: Colors.grey)), const SizedBox(height: 8),
          Wrap(spacing: 8, runSpacing: 6, children: [
            FilterChip(label: const Text('全部'), selected: _areaFilter.isEmpty, onSelected: (_) { setSt(() => _areaFilter = ''); Navigator.pop(ctx); setState(() {}); }),
            ..._areas.map((a) => FilterChip(label: Text(a['name']!), selected: _areaFilter == a['name'], onSelected: (_) { setSt(() { _areaFilter = a['name']!; _areaPattern = a['pattern']!; }); Navigator.pop(ctx); setState(() {}); })),
          ]), const SizedBox(height: 20),
        ],
        if (_algos.isNotEmpty) ...[const Text('按算法', style: TextStyle(fontSize: 14, color: Colors.grey)), const SizedBox(height: 8),
          Wrap(spacing: 8, runSpacing: 6, children: [
            FilterChip(label: const Text('全部'), selected: _algoFilter.isEmpty, onSelected: (_) { setSt(() => _algoFilter = ''); Navigator.pop(ctx); setState(() {}); }),
            ..._algos.map((a) => FilterChip(label: Text(a), selected: _algoFilter == a, onSelected: (_) { setSt(() => _algoFilter = a); Navigator.pop(ctx); setState(() {}); })),
          ]),
        ],
        const SizedBox(height: 10),
      ])),
    ));
  }

  Widget _rangeChip(String label, String value) {
    final selected = _range == value;
    return FilterChip(
      label: Text(label, style: TextStyle(fontSize: 12, color: selected ? Colors.white : Colors.grey.shade700)),
      selected: selected,
      selectedColor: Colors.blue,
      checkmarkColor: Colors.white,
      onSelected: (_) => _switchRange(value),
      visualDensity: VisualDensity.compact,
      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
    );
  }

  void _navOrders(int tab) => widget.onNavigateOrders?.call(tab);

  @override
  Widget build(BuildContext context) {
    final filtered = _filteredAlarms;
    return Scaffold(
      appBar: AppBar(title: const Text('WDOS 工作台'), actions: [
        IconButton(icon: const Icon(Icons.filter_list), onPressed: _showFilter, tooltip: '筛选'),
        IconButton(icon: const Icon(Icons.refresh), onPressed: _load),
      ]),
      body: RefreshIndicator(onRefresh: _load, child: ListView(padding: const EdgeInsets.all(12), children: [
        // 时间范围切换
        Row(children: [
          _rangeChip('24h', 'today'),
          const SizedBox(width: 6),
          _rangeChip('7天', 'week'),
          const SizedBox(width: 6),
          _rangeChip('30天', 'month'),
        ]),
        const SizedBox(height: 10),
        // 统计卡片 — 可点击跳转
        Row(children: [
          _statCard('待接单', _pending, Colors.red, Icons.warning_amber, () => _navOrders(0)),
          const SizedBox(width: 10),
          _statCard('处理中', _processing, Colors.orange, Icons.pending_actions, () => _navOrders(1)),
        ]),
        const SizedBox(height: 10),
        Row(children: [
          _statCard('已完成', _done, Colors.green, Icons.check_circle, () => _navOrders(2)),
          const SizedBox(width: 10),
          _statCard('已超时', _overtimePending, Colors.red.shade900, Icons.timer_off, () => _navOrders(0)),
        ]),
        const SizedBox(height: 12),

        if (_areaFilter.isNotEmpty || _algoFilter.isNotEmpty)
          Padding(padding: const EdgeInsets.only(bottom: 8), child: Row(children: [
            Text('筛选: ', style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
            if (_areaFilter.isNotEmpty) Chip(label: Text(_areaFilter, style: const TextStyle(fontSize: 12)), onDeleted: () => setState(() { _areaFilter = ''; _areaPattern = ''; }), materialTapTargetSize: MaterialTapTargetSize.shrinkWrap, visualDensity: VisualDensity.compact),
            if (_algoFilter.isNotEmpty) Chip(label: Text(_algoFilter, style: const TextStyle(fontSize: 12)), onDeleted: () => setState(() => _algoFilter = ''), materialTapTargetSize: MaterialTapTargetSize.shrinkWrap, visualDensity: VisualDensity.compact),
          ])),

        Text('${filtered.length} 条工单', style: TextStyle(color: Colors.grey.shade600, fontSize: 13)),
        const SizedBox(height: 4),

        ...filtered.map((a) {
          final degree = a['degree'] ?? 0; final dup = a['duplicate_count'] ?? 1; final picUrl = a['alarm_pic_url']?.toString() ?? '';
          final status = a['status']?.toString() ?? '';
          return Card(
            margin: const EdgeInsets.only(bottom: 8),
            child: InkWell(
              onTap: () => showOrderDetail(context, order: a, users: _users, onChanged: _load),
              borderRadius: BorderRadius.circular(12),
              child: Padding(padding: const EdgeInsets.all(12), child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                ClipRRect(borderRadius: BorderRadius.circular(6), child: picUrl.isNotEmpty
                    ? Image.network(ApiConfig.imageUrl(picUrl), width: 64, height: 48, fit: BoxFit.cover, errorBuilder: (_, __, ___) => _noImg(degree))
                    : _noImg(degree)),
                const SizedBox(width: 10),
                Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Row(children: [
                    Container(padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2), decoration: BoxDecoration(color: _degreeColor(degree).withOpacity(0.15), borderRadius: BorderRadius.circular(4)), child: Text(_degreeLabel(degree), style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: _degreeColor(degree)))),
                    const SizedBox(width: 4),
                    _statusBadge(status),
                  ]),
                  const SizedBox(height: 4),
                  Text(a['title'] ?? '', style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14), maxLines: 2, overflow: TextOverflow.ellipsis),
                  const SizedBox(height: 3),
                  Row(children: [
                    Text('📍 ${a['camera_name'] ?? ''}', style: const TextStyle(fontSize: 11, color: Colors.grey)),
                    const SizedBox(width: 6),
                    Text(_timeAgo(a['created_at']?.toString()), style: const TextStyle(fontSize: 11, color: Colors.grey)),
                    if (dup > 1) ...[const SizedBox(width: 6), Text('重复${dup}次', style: TextStyle(fontSize: 11, color: Colors.orange.shade800))],
                  ]),
                ])),
              ])),
            ),
          );
        }),
      ])),
    );
  }

  Widget _statCard(String label, int count, Color color, IconData icon, VoidCallback onTap) {
    return Expanded(
      child: Card(
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(12),
          child: Padding(padding: const EdgeInsets.all(14), child: Row(children: [
            Icon(icon, color: color, size: 32),
            const SizedBox(width: 10),
            Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text('$count', style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold, color: color)),
              Text(label, style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
            ]),
          ])),
        ),
      ),
    );
  }
}
