import 'dart:convert';
import 'package:flutter/material.dart';
import 'dart:html' as html;
import '../services/api.dart';
import '../config/api.dart';

/// 共享工单详情/操作页面（全屏）
class OrderDetailPage extends StatefulWidget {
  final Map<String, dynamic> order;
  final List<Map<String, dynamic>> users;
  final VoidCallback onChanged;

  const OrderDetailPage({
    super.key,
    required this.order,
    required this.users,
    required this.onChanged,
  });

  @override
  State<OrderDetailPage> createState() => _OrderDetailPageState();
}

class _OrderDetailPageState extends State<OrderDetailPage> {
  final _reasonCtrl = TextEditingController();
  bool _isFalseAlarm = false;
  bool _submitting = false;
  List<String> _attachments = [];
  bool _uploading = false;

  Map<String, dynamic> get o => widget.order;
  List<Map<String, dynamic>> get users => widget.users;

  Color _degColor(dynamic d) =>
      [Colors.grey, Colors.blue, Colors.orange, Colors.red, Colors.deepPurple][(d is int ? d : 0).clamp(0, 4)];
  String _degLabel(dynamic d) =>
      ['0', 'Ⅰ', 'Ⅱ', 'Ⅲ', 'Ⅳ'][(d is int ? d : 0).clamp(0, 4)];
  String _statusLabel(String s) =>
      {'pending': '待接单', 'processing': '处理中', 'completed': '已完成'}[s] ?? s;

  Future<bool> _confirm(String title, String msg) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: Text(msg),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('确认')),
        ],
      ),
    );
    return ok == true;
  }

  Future<void> _accept() async {
    if (!await _confirm('确认接单', '确定要接这个工单吗？')) return;
    try {
      await ApiService.post('/work-orders/${o['id']}/accept', {});
      widget.onChanged();
      if (mounted) Navigator.pop(context);
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('接单成功')));
    } on ApiException catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.message)));
    }
  }

  Future<void> _submit() async {
    final resolution = _isFalseAlarm ? '错误报警' : _reasonCtrl.text;
    if (!_isFalseAlarm && resolution.trim().isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请输入处理说明')));
      return;
    }
    if (!await _confirm('确认提交', _isFalseAlarm ? '确认为错误报警并提交？' : '确定提交处理结果？')) return;
    setState(() => _submitting = true);
    try {
      await ApiService.post('/work-orders/${o['id']}/submit', {
        'resolution': resolution,
        'form_data': jsonEncode({'attachments': _attachments}),
        'proof_images': _attachments.join(','),
      });
      widget.onChanged();
      if (mounted) Navigator.pop(context);
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('提交成功')));
    } on ApiException catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.message)));
    }
    setState(() => _submitting = false);
  }

  Future<void> _pickAndUpload() async {
    final input = html.FileUploadInputElement()..accept = 'image/*';
    input.click();
    input.onChange.listen((e) async {
      final files = input.files;
      if (files == null || files.isEmpty) return;
      setState(() => _uploading = true);
      for (final file in files) {
        try {
          final reader = html.FileReader();
          reader.readAsArrayBuffer(file);
          await reader.onLoad.first;
          final token = await ApiConfig.getToken();
          final xhr = html.HttpRequest();
          xhr.open('POST', '${ApiConfig.baseUrl}/upload');
          xhr.setRequestHeader('Authorization', 'Bearer $token');
          final formData = html.FormData();
          formData.appendBlob('file', file);
          xhr.send(formData);
          await xhr.onLoad.first;
          if (xhr.status == 200) {
            final resp = jsonDecode(xhr.responseText!);
            final url = resp['data']['url']?.toString() ?? '';
            if (url.isNotEmpty) setState(() => _attachments.add(url));
          }
        } catch (_) {}
      }
      setState(() => _uploading = false);
    });
  }

  void _transfer() {
    int? selectedUserId;
    final reasonCtrl = TextEditingController();
    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(builder: (ctx, setSt) => AlertDialog(
        title: const Text('转交工单'),
        content: Column(mainAxisSize: MainAxisSize.min, children: [
          DropdownButtonFormField<int>(
            decoration: const InputDecoration(labelText: '选择目标用户', border: OutlineInputBorder()),
            items: users.where((u) => u['status'] == 'active').map((u) => DropdownMenuItem(value: u['id'] as int, child: Text('${u['name']} (${u['role']})'))).toList(),
            onChanged: (v) => setSt(() => selectedUserId = v),
            value: selectedUserId,
          ),
          const SizedBox(height: 8),
          TextField(controller: reasonCtrl, decoration: const InputDecoration(labelText: '转交原因', border: OutlineInputBorder())),
        ]),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('取消')),
          FilledButton(onPressed: selectedUserId == null ? null : () async {
            try {
              final target = users.firstWhere((u) => u['id'] == selectedUserId);
              await ApiService.post('/work-orders/${o['id']}/transfer', {
                'transfer_to_user_id': selectedUserId,
                'transfer_to_user_name': target['name'] ?? '',
                'reason': reasonCtrl.text,
              });
              Navigator.pop(ctx);
              widget.onChanged();
              if (this.mounted) Navigator.pop(this.context);
              if (mounted) ScaffoldMessenger.of(this.context).showSnackBar(const SnackBar(content: Text('转交成功')));
            } on ApiException catch (e) {
              if (mounted) ScaffoldMessenger.of(this.context).showSnackBar(SnackBar(content: Text(e.message)));
            }
          }, child: const Text('确认转交')),
        ],
      )),
    );
  }

  @override
  Widget build(BuildContext context) {
    final status = o['status']?.toString() ?? '';
    final picUrl = o['alarm_pic_url']?.toString() ?? '';
    final dup = o['duplicate_count'] ?? 1;
    final degree = o['degree'];
    final assignee = o['assignee_name']?.toString();
    final accepter = o['accepter_name']?.toString();

    return Scaffold(
      appBar: AppBar(
        title: const Text('工单详情'),
      ),
      body: ListView(padding: const EdgeInsets.all(16), children: [
        // 等级 + 状态
        Wrap(spacing: 8, runSpacing: 6, children: [
          Container(padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3), decoration: BoxDecoration(color: _degColor(degree).withOpacity(0.15), borderRadius: BorderRadius.circular(4)), child: Text(_degLabel(degree), style: TextStyle(fontWeight: FontWeight.bold, color: _degColor(degree)))),
          Container(padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2), decoration: BoxDecoration(color: _statusColor(status).withOpacity(0.1), borderRadius: BorderRadius.circular(4)), child: Text(_statusLabel(status), style: TextStyle(fontSize: 11, fontWeight: FontWeight.w500, color: _statusColor(status)))),
          if (dup > 1) Container(padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2), decoration: BoxDecoration(color: Colors.orange.shade50, borderRadius: BorderRadius.circular(4)), child: Text('重复${dup}次', style: TextStyle(fontSize: 11, color: Colors.orange.shade800))),
        ]),
        const SizedBox(height: 12),
        Text(o['title'] ?? '', style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
        const SizedBox(height: 16),

        // 信息
        _row('编号', o['order_no']),
        _row('地点', o['camera_name']),
        _row('算法', o['algorithm_name']),
        _row('指派人', (assignee != null && assignee.isNotEmpty) ? assignee : '待分配'),
        if (accepter != null && accepter.isNotEmpty) _row('处理人', accepter),
        if (o['department_name'] != null && o['department_name'].toString().isNotEmpty)
          _row('部门', o['department_name']),
        if (o['resolution'] != null && o['resolution'].toString().isNotEmpty) ...[
          const Divider(height: 24),
          _row('处理结果', o['resolution']),
        ],

        // 报警图片
        if (picUrl.isNotEmpty) ...[
          const SizedBox(height: 12),
          const Text('报警图片', style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600)),
          const SizedBox(height: 6),
          GestureDetector(
            onTap: () => Navigator.push(context, MaterialPageRoute(
              builder: (_) => Scaffold(appBar: AppBar(title: const Text('报警图片')), body: Center(child: InteractiveViewer(child: Image.network(ApiConfig.imageUrl(picUrl), fit: BoxFit.contain)))),
            )),
            child: ClipRRect(borderRadius: BorderRadius.circular(8), child: Image.network(ApiConfig.imageUrl(picUrl), width: double.infinity, fit: BoxFit.cover, errorBuilder: (_, __, ___) => Container(height: 160, color: Colors.grey.shade100, child: const Center(child: Text('加载失败'))))),
          ),
        ],

        // 已完成：附件
        if (status == 'completed') _buildAttachments(),

        // 待接单：醒目接单按钮
        if (status == 'pending') ...[
          const SizedBox(height: 24),
          SizedBox(
            width: double.infinity,
            height: 48,
            child: FilledButton.icon(
              onPressed: _accept,
              icon: const Icon(Icons.check_circle, size: 22),
              label: const Text('接单', style: TextStyle(fontSize: 17)),
            ),
          ),
        ],

        // 处理中：提交表单
        if (status == 'processing') ...[
          const SizedBox(height: 20),
          const Divider(),
          const SizedBox(height: 8),
          const Text('处理工单', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
          const SizedBox(height: 10),
          Row(children: [
            SizedBox(width: 24, height: 24, child: Checkbox(value: _isFalseAlarm, onChanged: (v) => setState(() => _isFalseAlarm = v ?? false))),
            const SizedBox(width: 8),
            const Text('错误报警'),
          ]),
          if (!_isFalseAlarm) ...[
            const SizedBox(height: 10),
            TextField(controller: _reasonCtrl, decoration: const InputDecoration(labelText: '处理说明', hintText: '描述处理过程', border: OutlineInputBorder()), maxLines: 3),
            const SizedBox(height: 10),
            Row(children: [
              OutlinedButton.icon(onPressed: _uploading ? null : _pickAndUpload, icon: _uploading ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2)) : const Icon(Icons.add_photo_alternate, size: 18), label: const Text('上传图片')),
              const SizedBox(width: 8),
              Text('${_attachments.length} 张', style: const TextStyle(color: Colors.grey, fontSize: 12)),
            ]),
            if (_attachments.isNotEmpty) ...[
              const SizedBox(height: 8),
              SizedBox(height: 72, child: ListView.separated(scrollDirection: Axis.horizontal, itemCount: _attachments.length, separatorBuilder: (_, __) => const SizedBox(width: 6), itemBuilder: (_, i) => ClipRRect(borderRadius: BorderRadius.circular(6), child: Image.network(ApiConfig.imageUrl(_attachments[i]), width: 72, height: 72, fit: BoxFit.cover, errorBuilder: (_, __, ___) => Container(width: 72, height: 72, color: Colors.grey.shade100, child: const Icon(Icons.broken_image))))),
              ),
            ],
          ],
          const SizedBox(height: 16),
          SizedBox(width: double.infinity, child: FilledButton(onPressed: _submitting ? null : _submit, child: _submitting ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white)) : const Text('提交处理'))),
        ],

        const SizedBox(height: 30),
      ]),

      // 底部操作栏：转交
      bottomNavigationBar: status == 'pending' || status == 'processing'
          ? SafeArea(child: Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
              child: Row(children: [
                Expanded(child: OutlinedButton.icon(onPressed: _transfer, icon: const Icon(Icons.swap_horiz, size: 18), label: const Text('转交工单'))),
              ]),
            ))
          : null,
    );
  }

  Color _statusColor(String s) =>
      {'pending': Colors.red, 'processing': Colors.orange, 'completed': Colors.green}[s] ?? Colors.grey;

  Widget _row(String l, dynamic v) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 4),
    child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
      SizedBox(width: 60, child: Text(l, style: const TextStyle(color: Colors.grey, fontSize: 13))),
      Expanded(child: Text('$v', style: const TextStyle(fontSize: 13))),
    ]),
  );

  Widget _buildAttachments() {
    List<String> proofImages = [];
    try {
      final fd = o['form_data'];
      if (fd != null && fd is String && fd.isNotEmpty) {
        final parsed = jsonDecode(fd);
        if (parsed is Map && parsed['attachments'] is List) {
          proofImages = List<String>.from(parsed['attachments']);
        }
      }
    } catch (_) {}
    if (o['proof_images'] != null && o['proof_images'].toString().isNotEmpty) {
      final pis = o['proof_images'].toString().split(',').where((s) => s.trim().isNotEmpty).toList();
      if (pis.isNotEmpty) proofImages = pis;
    }
    if (proofImages.isEmpty) return const SizedBox.shrink();
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      const SizedBox(height: 10),
      const Divider(),
      const SizedBox(height: 6),
      const Text('处理附件', style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600)),
      const SizedBox(height: 6),
      SizedBox(height: 100, child: ListView.separated(scrollDirection: Axis.horizontal, itemCount: proofImages.length, separatorBuilder: (_, __) => const SizedBox(width: 8), itemBuilder: (_, i) => GestureDetector(
        onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => Scaffold(appBar: AppBar(title: Text('附件 ${i + 1}')), body: Center(child: InteractiveViewer(child: Image.network(ApiConfig.imageUrl(proofImages[i]), fit: BoxFit.contain)))))),
        child: ClipRRect(borderRadius: BorderRadius.circular(6), child: Image.network(ApiConfig.imageUrl(proofImages[i]), width: 100, height: 100, fit: BoxFit.cover, errorBuilder: (_, __, ___) => Container(width: 100, height: 100, color: Colors.grey.shade100, child: const Icon(Icons.broken_image)))),
      ))),
    ]);
  }
}

/// 便捷调用：打开全屏工单详情页
void showOrderDetail(BuildContext ctx, {
  required Map<String, dynamic> order,
  required List<Map<String, dynamic>> users,
  required VoidCallback onChanged,
}) {
  Navigator.of(ctx).push(MaterialPageRoute(
    builder: (_) => OrderDetailPage(order: order, users: users, onChanged: onChanged),
  ));
}
