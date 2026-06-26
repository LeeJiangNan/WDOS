import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'dart:convert';
import '../services/api.dart';
import '../config/api.dart';
import 'login.dart';

class ProfilePage extends StatefulWidget {
  final VoidCallback? onGoToOrders;
  const ProfilePage({super.key, this.onGoToOrders});
  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  // 当前用户
  String _userName = '', _userRole = '', _userDept = '';
  int _userId = 0;

  // 统计数据
  int _done = 0, _pending = 0, _processing = 0, _overtime = 0;

  // 身份切换
  List<Map<String, dynamic>> _subordinates = [];
  List<Map<String, dynamic>> _departments = [];
  int? _viewAsId;
  String _viewAsName = '';

  // 排班
  Map<String, String> _weekSchedule = {};
  bool _showSchedule = false;

  // 统计范围
  String _range = 'today';
  final Map<String, Map<String, int>> _statsCache = {};  // 缓存防卡

  @override
  void initState() {
    super.initState();
    _loadUser();
    _loadStats();
    _loadSubordinates();
    _loadDepartments();
  }

  // ========== 数据加载 ==========

  Future<void> _loadUser() async {
    final prefs = await SharedPreferences.getInstance();
    final userJson = prefs.getString('wdos_user');
    if (userJson != null) {
      final user = jsonDecode(userJson);
      setState(() {
        _userName = user['name'] ?? '用户';
        _userRole = user['role'] ?? '';
        _userId = user['id'] ?? 0;
      });
    }
    // 从 API 刷新
    try {
      final data = await ApiService.get('/users');
      final list = List<Map<String, dynamic>>.from(data['list'] ?? []);
      for (final u in list) {
        if (u['id'] == _userId) {
          setState(() {
            _userName = u['name']?.toString() ?? _userName;
            _userRole = u['role']?.toString() ?? _userRole;
            _userDept = u['department_name']?.toString() ?? '';
          });
          await prefs.setString('wdos_user', jsonEncode(u));
          break;
        }
      }
    } catch (_) {}
  }

  Future<void> _loadStats() async {
    // 先用缓存
    final cached = _statsCache[_range];
    if (cached != null) {
      setState(() {
        _pending = cached['pending'] ?? 0;
        _processing = cached['processing'] ?? 0;
        _done = cached['done'] ?? 0;
        _overtime = cached['overtime'] ?? 0;
      });
    }
    final ep = _range == 'week'
        ? '/stats/weekly-overview'
        : _range == 'month'
            ? '/stats/monthly-overview'
            : '/stats/my-overview';
    try {
      final data = await ApiService.get(ep);
      final p = data['pending_orders'] ?? 0;
      final r = data['processing_orders'] ?? 0;
      final d = data['completed_orders'] ?? 0;
      final o = data['overtime_orders'] ?? 0;
      _statsCache[_range] = {'pending': p, 'processing': r, 'done': d, 'overtime': o};
      if (mounted) setState(() { _pending = p; _processing = r; _done = d; _overtime = o; });
    } catch (_) {}
  }

  Future<void> _loadSubordinates() async {
    if (_userRole == 'handler') return;
    try {
      final data = await ApiService.get('/users/subordinates');
      setState(() {
        _subordinates =
            List<Map<String, dynamic>>.from(data['list'] ?? []);
      });
    } catch (_) {}
  }

  Future<void> _loadDepartments() async {
    if (_userRole != 'admin' && _userRole != 'director') return;
    try {
      final data = await ApiService.get('/departments');
      setState(() {
        _departments =
            List<Map<String, dynamic>>.from(data['list'] ?? data['data'] ?? []);
      });
    } catch (_) {}
  }

  Future<void> _loadSchedule() async {
    try {
      final now = DateTime.now();
      final monday = now.subtract(Duration(days: now.weekday - 1));
      final schedule = <String, String>{};
      for (int i = 0; i < 7; i++) {
        final date = monday.add(Duration(days: i));
        final dateStr =
            '${date.year}-${date.month.toString().padLeft(2, '0')}-${date.day.toString().padLeft(2, '0')}';
        try {
          final data = await ApiService.get('/schedules?date=$dateStr');
          if (data is Map) {
            for (final entry in data.entries) {
              if (entry.value is List) {
                for (final s in entry.value) {
                  if (s is Map && s['user_id'] == _userId) {
                    final shift = s['shift_type'] == 'day' ? '☀️' : '🌙';
                    final area = (s['area']?.toString() ?? '').isNotEmpty
                        ? ' ${s['area']}'
                        : '';
                    schedule[dateStr] = '$shift$area';
                  }
                }
              }
            }
          }
        } catch (_) {}
      }
      setState(() {
        _weekSchedule = schedule;
        _showSchedule = true;
      });
    } catch (_) {
      setState(() => _showSchedule = true);
    }
  }

  // ========== 身份切换 ==========

  void _switchIdentity(int? userId, String name) {
    setState(() {
      if (userId == null) {
        _viewAsId = null;
        _viewAsName = '';
        ApiConfig.viewAsUserId = null;
      } else {
        _viewAsId = userId;
        _viewAsName = name;
        ApiConfig.viewAsUserId = userId;
      }
    });
    _loadStats();
  }

  // ========== 下钻视图 ==========

  void _showDrillDown() {
    final role = _viewAsId != null
        ? _subordinates
            .firstWhere((s) => s['id'] == _viewAsId, orElse: () => {})['role']
            ?.toString() ?? ''
        : _userRole;

    // 一线人员：只显示自己的数据
    if (role == 'handler' && _viewAsId == null) {
      _showSimpleStats();
      return;
    }

    // 管理部/总监：先选部门
    if (_userRole == 'admin' || _userRole == 'director') {
      _showDeptPicker();
      return;
    }

    // 领班/经理：直接看部门数据，可切换到个人
    _showDeptStats();
  }

  void _showSimpleStats() {
    showDialog(
      context: context,
      builder: (_) => AlertDialog(
        title: Text('$_userName 的数据'),
        content: Column(mainAxisSize: MainAxisSize.min, children: [
          _statRow('待接单', _pending, Colors.red),
          _statRow('处理中', _processing, Colors.orange),
          _statRow('已完成', _done, Colors.green),
          _statRow('超时', _overtime, Colors.red.shade900),
        ]),
        actions: [
          FilledButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('关闭'))
        ],
      ),
    );
  }

  void _showDeptPicker() {
    showModalBottomSheet(
      context: context,
      builder: (ctx) => Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('选择部门', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
            const SizedBox(height: 16),
            if (_departments.isEmpty)
              const Center(child: Text('暂无部门数据'))
            else
              ..._departments.map((d) => ListTile(
                    leading: const Icon(Icons.business, color: Colors.blue),
                    title: Text(d['name']?.toString() ?? ''),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () {
                      Navigator.pop(ctx);
                      _showDeptStatsForDept(
                          d['id'] as int?, d['name']?.toString() ?? '');
                    },
                  )),
          ],
        ),
      ),
    );
  }

  void _showDeptStats() {
    // 领班/经理：显示部门统计 + 下级列表
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => DraggableScrollableSheet(
        initialChildSize: 0.7,
        minChildSize: 0.4,
        maxChildSize: 0.9,
        expand: false,
        builder: (ctx, scrollCtrl) => _deptStatsContent(scrollCtrl, null, null),
      ),
    );
  }

  void _showDeptStatsForDept(int? deptId, String deptName) {
    // 管理部选中某部门后的视图
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => DraggableScrollableSheet(
        initialChildSize: 0.7,
        minChildSize: 0.4,
        maxChildSize: 0.9,
        expand: false,
        builder: (ctx, scrollCtrl) =>
            _deptStatsContent(scrollCtrl, deptId, deptName),
      ),
    );
  }

  Widget _deptStatsContent(
      ScrollController scrollCtrl, int? deptId, String? deptName) {
    // 过滤该部门的下级
    final members = _subordinates.where((s) {
      if (deptId == null) return true;
      return s['department_id'] == deptId;
    }).toList();

    return ListView(
      controller: scrollCtrl,
      padding: const EdgeInsets.all(20),
      children: [
        Center(
          child: Container(
            width: 40, height: 4,
            decoration: BoxDecoration(color: Colors.grey.shade300, borderRadius: BorderRadius.circular(2)),
          ),
        ),
        const SizedBox(height: 16),
        Text(
          deptName ?? '部门数据统计',
          style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
        ),
        Text(
          '当前查看: ${_viewAsId != null ? _viewAsName : "部门整体"}',
          style: const TextStyle(color: Colors.grey, fontSize: 13),
        ),
        const SizedBox(height: 16),
        // 统计卡片
        Row(children: [
          _miniStat('待接单', '$_pending', Colors.red),
          const SizedBox(width: 8),
          _miniStat('处理中', '$_processing', Colors.orange),
          const SizedBox(width: 8),
          _miniStat('已完成', '$_done', Colors.green),
          const SizedBox(width: 8),
          _miniStat('超时', '$_overtime', Colors.red.shade900),
        ]),
        const SizedBox(height: 8),
        // 范围切换
        Row(children: [
          _rangeChip('24h', 'today'),
          const SizedBox(width: 6),
          _rangeChip('7天', 'week'),
          const SizedBox(width: 6),
          _rangeChip('30天', 'month'),
        ]),
        const SizedBox(height: 20),
        // 切回部门整体或自己
        if (_viewAsId != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: Row(children: [
              OutlinedButton.icon(
                onPressed: () => _switchIdentity(null, ''),
                icon: const Icon(Icons.undo, size: 16),
                label: const Text('切回部门整体'),
              ),
              const SizedBox(width: 8),
              OutlinedButton.icon(
                onPressed: () {
                  _switchIdentity(null, '');
                },
                icon: const Icon(Icons.person, size: 16),
                label: const Text('看我自己的'),
              ),
            ]),
          ),
        // 人员列表
        if (members.isNotEmpty) ...[
          const Text('查看个人数据', style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600)),
          const SizedBox(height: 8),
          ...members.map((s) => ListTile(
                leading: CircleAvatar(
                  backgroundColor: _viewAsId == s['id']
                      ? Colors.blue
                      : Colors.blue.shade50,
                  child: Text(
                    s['name']?.toString().substring(0, 1) ?? '?',
                    style: TextStyle(
                        color: _viewAsId == s['id']
                            ? Colors.white
                            : Colors.blue),
                  ),
                ),
                title: Text(s['name']?.toString() ?? ''),
                subtitle: Text(_roleLabel(s['role']?.toString())),
                trailing: _viewAsId == s['id']
                    ? const Icon(Icons.visibility, color: Colors.blue, size: 20)
                    : null,
                onTap: () {
                  _switchIdentity(
                      s['id'] as int?, s['name']?.toString() ?? '');
                  Navigator.pop(context);
                  // 自动切换到个人统计
                  Future.delayed(const Duration(milliseconds: 300), () {
                    _showDrillDown();
                  });
                },
              )),
        ] else
          const Center(
            child: Padding(
              padding: EdgeInsets.all(20),
              child: Text('暂无下级人员', style: TextStyle(color: Colors.grey)),
            ),
          ),
      ],
    );
  }

  // ========== 退出 ==========

  Future<void> _logout() async {
    await ApiConfig.clearToken();
    if (mounted) {
      Navigator.pushAndRemoveUntil(context,
          MaterialPageRoute(builder: (_) => const LoginPage()),
          (route) => false);
    }
  }

  // ========== UI 组件 ==========

  Widget _rangeChip(String label, String value) {
    final selected = _range == value;
    return FilterChip(
      label: Text(label,
          style: TextStyle(
              fontSize: 12,
              color: selected ? Colors.white : Colors.grey.shade700)),
      selected: selected,
      selectedColor: Colors.blue,
      checkmarkColor: Colors.white,
      onSelected: (_) {
        setState(() => _range = value);
        _loadStats();
      },
      visualDensity: VisualDensity.compact,
      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
    );
  }

  Widget _miniStat(String label, String value, Color color) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 10),
        decoration: BoxDecoration(
          color: color.withOpacity(0.08),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Column(children: [
          Text(value,
              style:
                  TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: color)),
          const SizedBox(height: 2),
          Text(label, style: const TextStyle(fontSize: 10, color: Colors.grey)),
        ]),
      ),
    );
  }

  String _dayOfWeek(String dateStr) {
    final d = DateTime.tryParse(dateStr);
    if (d == null) return '';
    return ['一', '二', '三', '四', '五', '六', '日'][d.weekday - 1];
  }

  String _roleLabel(String? role) {
    return {
      'admin': '管理员',
      'director': '总监',
      'manager': '经理',
      'supervisor': '领班',
      'handler': '一线人员'
    }[role ?? ''] ??
        role ??
        '';
  }

  Widget _statRow(String label, int value, Color color) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
        Text(label, style: const TextStyle(fontSize: 14)),
        Text('$value',
            style: TextStyle(
                fontSize: 16, fontWeight: FontWeight.bold, color: color)),
      ]),
    );
  }

  // ========== 主页面 ==========

  @override
  Widget build(BuildContext context) {
    final roleLabel = _roleLabel(_userRole);
    final isLeader = _userRole == 'supervisor' ||
        _userRole == 'manager' ||
        _userRole == 'admin' ||
        _userRole == 'director';
    final isDeptAdmin = _userRole == 'admin' || _userRole == 'director';

    return Scaffold(
      appBar: AppBar(title: const Text('我的')),
      body: RefreshIndicator(
        onRefresh: () async {
          _loadStats();
          _loadSubordinates();
        },
        child: ListView(padding: const EdgeInsets.all(14), children: [
          // 用户信息卡片
          Card(
            color: Colors.blue.shade700,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(children: [
                const CircleAvatar(
                    radius: 26,
                    backgroundColor: Colors.white24,
                    child: Icon(Icons.person, color: Colors.white)),
                const SizedBox(width: 14),
                Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(_userName,
                          style: const TextStyle(
                              fontSize: 18,
                              fontWeight: FontWeight.w600,
                              color: Colors.white)),
                      const SizedBox(height: 2),
                      Text(roleLabel,
                          style: const TextStyle(
                              color: Colors.white70, fontSize: 13)),
                    ]),
                const Spacer(),
                if (isLeader && _viewAsId != null)
                  TextButton(
                    onPressed: () => _switchIdentity(null, ''),
                    child:
                        const Text('切回自己', style: TextStyle(color: Colors.white)),
                  ),
              ]),
            ),
          ),

          const SizedBox(height: 10),

          // 查看提示
          if (_viewAsId != null)
            Card(
              color: Colors.orange.shade50,
              child: ListTile(
                dense: true,
                leading: const Icon(Icons.visibility, color: Colors.orange),
                title: Text('正在查看: $_viewAsName',
                    style: const TextStyle(fontSize: 13)),
                trailing: TextButton(
                  onPressed: () => _switchIdentity(null, ''),
                  child: const Text('退出'),
                ),
              ),
            ),

          const SizedBox(height: 8),

          // 时间范围切换
          Row(children: [
            _rangeChip('24h', 'today'),
            const SizedBox(width: 6),
            _rangeChip('7天', 'week'),
            const SizedBox(width: 6),
            _rangeChip('30天', 'month'),
          ]),

          const SizedBox(height: 10),

          // 统计卡片 — 可点击下钻
          Row(children: [
            _statCard('待接单', _pending, Colors.red, Icons.warning_amber,
                isLeader ? _showDrillDown : _showSimpleStats),
            const SizedBox(width: 8),
            _statCard('已完成', _done, Colors.green, Icons.check_circle,
                isLeader ? _showDrillDown : _showSimpleStats),
          ]),
          const SizedBox(height: 8),
          Row(children: [
            _statCard('处理中', _processing, Colors.orange,
                Icons.pending_actions, isLeader ? _showDrillDown : _showSimpleStats),
            const SizedBox(width: 8),
            _statCard(
                '已超时',
                _overtime,
                Colors.red.shade900,
                Icons.timer_off,
                isLeader ? _showDrillDown : _showSimpleStats),
          ]),

          const SizedBox(height: 16),

          // 部门切换入口（管理部专用）
          if (isDeptAdmin && _departments.isNotEmpty) ...[
            Wrap(
              spacing: 6,
              runSpacing: 4,
              children: _departments.map((d) => ActionChip(
                    avatar: const Icon(Icons.business, size: 16),
                    label: Text(d['name']?.toString() ?? '',
                        style: const TextStyle(fontSize: 12)),
                    onPressed: () => _showDeptStatsForDept(
                        d['id'] as int?, d['name']?.toString() ?? ''),
                  )).toList(),
            ),
            const SizedBox(height: 16),
          ],

          // 菜单
          Card(
            child: Column(children: [
              ListTile(
                leading: const Icon(Icons.assignment),
                title: const Text('我的工单'),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => widget.onGoToOrders?.call(),
              ),
              ListTile(
                leading: const Icon(Icons.calendar_today),
                title: const Text('我的排班（本周）'),
                trailing: _showSchedule
                    ? const Icon(Icons.refresh, size: 18)
                    : const Icon(Icons.chevron_right),
                onTap: _loadSchedule,
                subtitle: _showSchedule
                    ? _weekSchedule.isEmpty
                        ? const Text('本周无排班',
                            style:
                                TextStyle(color: Colors.grey, fontSize: 12))
                        : Padding(
                            padding: const EdgeInsets.only(top: 6),
                            child: SingleChildScrollView(
                              scrollDirection: Axis.horizontal,
                              child: Row(
                                  children: _weekSchedule.entries.map((e) {
                                final day = e.key.substring(5);
                                final dow = _dayOfWeek(e.key);
                                return Container(
                                  width: 56,
                                  margin: const EdgeInsets.only(right: 4),
                                  padding:
                                      const EdgeInsets.symmetric(vertical: 4),
                                  decoration: BoxDecoration(
                                      color: Colors.blue.shade50,
                                      borderRadius: BorderRadius.circular(4)),
                                  child: Column(children: [
                                    Text(dow,
                                        style: TextStyle(
                                            fontSize: 9,
                                            color: Colors.grey.shade600)),
                                    Text(day,
                                        style: const TextStyle(
                                            fontSize: 10,
                                            fontWeight: FontWeight.bold)),
                                    Text(e.value,
                                        style: const TextStyle(fontSize: 12)),
                                  ]),
                                );
                              }).toList()),
                            ),
                          )
                    : const Text('点击加载',
                        style: TextStyle(color: Colors.blue, fontSize: 12)),
              ),
            ]),
          ),
          const SizedBox(height: 16),
          FilledButton.tonal(
              onPressed: _logout, child: const Text('退出登录')),
          const SizedBox(height: 40),
        ]),
      ),
    );
  }

  Widget _statCard(
      String label, int count, Color color, IconData icon, VoidCallback onTap) {
    return Expanded(
      child: Card(
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(12),
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: Row(children: [
              Icon(icon, color: color, size: 32),
              const SizedBox(width: 10),
              Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('$count',
                    style: TextStyle(
                        fontSize: 24,
                        fontWeight: FontWeight.bold,
                        color: color)),
                Text(label,
                    style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
              ]),
            ]),
          ),
        ),
      ),
    );
  }
}
