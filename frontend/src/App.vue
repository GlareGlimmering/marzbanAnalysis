<template>
  <div class="dashboard-root">

    <!-- 头部区域 -->
    <header class="dashboard-header">
      <div class="header-title">
        <h1>Xray Monitor</h1>
        <p>网络实时数据行为分析面板</p>
      </div>
      <div class="status-badge">
        <span class="pulse-dot"></span>
        Dispatcher Running
      </div>
    </header>

    <main class="dashboard-main">

      <!-- 核心指标卡片组 (原 Flutter Card 悬浮流) -->
      <div class="metrics-grid">
        <div class="flutter-card hover-lift shadow-blue">
          <div class="card-label">总处理请求数</div>
          <div class="card-value value-blue">{{ overview.total_requests || 0 }}</div>
        </div>

        <div class="flutter-card hover-lift shadow-purple">
          <div class="card-label">活跃代理用户</div>
          <div class="card-value value-purple">{{ overview.active_users || 0 }}</div>
        </div>

        <div class="flutter-card hover-lift shadow-emerald">
          <div class="card-label">活跃出口通道</div>
          <div class="card-value value-emerald">{{ overview.active_outbounds || 0 }}</div>
        </div>
      </div>

      <!-- 图表分栏 -->
      <div class="charts-grid">
        <div class="flutter-card">
          <h3 class="section-sub-title"><span class="dot blue"></span> 入站客户端来源排行</h3>
          <div id="inboundChart" class="chart-container"></div>
        </div>

        <div class="flutter-card">
          <h3 class="section-sub-title"><span class="dot emerald"></span> 落地线路负载排行</h3>
          <div id="outboundChart" class="chart-container"></div>
        </div>
      </div>

      <!-- 级联分析大面板 -->
      <section class="analysis-section">
        <div class="section-title-bar">
          <h2>客户端级联 analysis 面板</h2>
          <p>交互式联动：选择下方的代理用户及对应的物理源 IP，右侧磁贴将精确解构出其 Top 20 目标流量去向。</p>
        </div>

        <div class="split-panel">

          <!-- 左侧：用户目录卡片 -->
          <div class="flutter-card side-panel">
            <div class="panel-header">
              <h3>用户实体目录</h3>
              <p>点击展开用户绑定的活跃终端 IP</p>
            </div>

            <div class="list-container">
              <div v-for="user in userHierarchy" :key="user.email" class="user-group-item">
                <div
                    @click="toggleUser(user.email)"
                    :class="['user-row', { 'is-active': selectedEmail === user.email }]"
                >
                  <span class="email-text">{{ user.email }}</span>
                  <span class="ip-count-badge">{{ user.ips.length }} IP</span>
                </div>

                <div v-if="expandedUsers.has(user.email)" class="ip-dropdown-box">
                  <button
                      v-for="ip in user.ips"
                      :key="ip"
                      @click.stop="selectEntity(user.email, ip)"
                      :class="['ip-btn', { 'is-selected': selectedEmail === user.email && selectedIP === ip }]"
                  >
                    <span>{{ ip }}</span>
                    <span
                        title="复制 IP"
                        role="button"
                        aria-label="复制 IP"
                        @click.stop="copyInboundIP(ip)"
                        style="margin-left: auto; margin-right: 8px; font-size: 12px; opacity: 0.78; cursor: pointer; user-select: none;"
                    >
                      {{ copiedIP === ip ? '已复制' : '📋' }}
                    </span>
                    <span v-if="selectedEmail === user.email && selectedIP === ip" class="active-dot"></span>
                  </button>
                </div>
              </div>

              <div v-if="userHierarchy.length === 0" class="empty-state">
                暂无活跃用户级联数据
              </div>
            </div>
          </div>

          <!-- 右侧：详情表格卡片 -->
          <div class="flutter-card main-panel">
            <div class="panel-header flex-between">
              <div class="path-info">
                <h3>目标行为流向映射</h3>
                <div class="path-tags">
                  <span class="path-label">当前解构路径:</span>
                  <span v-if="selectedEmail" class="tag tag-blue">{{ selectedEmail }}</span>
                  <span v-if="selectedIP" class="tag tag-emerald">{{ selectedIP }}</span>
                  <span v-if="!selectedEmail" class="tag-none">未选择监听实体</span>
                </div>
              </div>
              <span class="top-badge">Top 20 频次</span>
            </div>

            <!-- 数据表格 -->
            <div class="table-wrapper">
              <table class="custom-table">
                <thead>
                <tr>
                  <th class="col-index">Index</th>
                  <th>Target Domain / Destination IP</th>
                  <th class="col-count">Hit Count</th>
                </tr>
                </thead>
                <tbody>
                <tr v-for="(detail, idx) in ipSpecificTargets" :key="idx" class="table-row">
                  <td class="cell-index">{{ idx + 1 }}</td>
                  <td class="cell-target">
                    <span class="target-chip">{{ detail.target }}</span>
                  </td>
                  <td class="cell-count highlight-text">{{ detail.count }} <span class="unit">hits</span></td>
                </tr>
                <tr v-if="ipSpecificTargets.length === 0">
                  <td colspan="3" class="table-empty">
                    暂无精确匹配流向数据，请选择左侧实体
                  </td>
                </tr>
                </tbody>
              </table>
            </div>
          </div>

        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, nextTick } from 'vue';
import * as echarts from 'echarts';

const overview = ref<any>({});
const userHierarchy = ref<any[]>([]);

const selectedEmail = ref<string>('');
const selectedIP = ref<string>('');
const expandedUsers = ref<Set<string>>(new Set());
const ipSpecificTargets = ref<any[]>([]);
const copiedIP = ref<string>('');

const rawInboundData = ref<any[]>([]);
const rawOutboundData = ref<any[]>([]);

let inboundChart: echarts.ECharts | null = null;
let outboundChart: echarts.ECharts | null = null;

const toggleUser = (email: string) => {
  if (expandedUsers.value.has(email)) {
    expandedUsers.value.delete(email);
  } else {
    expandedUsers.value.add(email);
  }
};

const selectEntity = async (email: string, ip: string) => {
  selectedEmail.value = email;
  selectedIP.value = ip;
  try {
    const res = await fetch(`/data/api/ip-targets?email=${encodeURIComponent(email)}&ip=${encodeURIComponent(ip)}`);
    if (res.ok) {
      ipSpecificTargets.value = await res.json();
    }
  } catch (err) {
    console.error('联合数据抓取异常:', err);
  }
};

const copyInboundIP = async (ip: string) => {
  try {
    await navigator.clipboard.writeText(ip);
    copiedIP.value = ip;
    window.setTimeout(() => {
      if (copiedIP.value === ip) copiedIP.value = '';
    }, 1200);
  } catch (err) {
    console.error('复制 IP 失败:', err);
  }
};

const fetchData = async () => {
  try {
    const overRes = await fetch('/data/api/overview');
    if (overRes.ok) overview.value = await overRes.json();

    const hierarchyRes = await fetch('/data/api/user-hierarchy');
    if (hierarchyRes.ok) {
      const hierarchyData = await hierarchyRes.json();
      userHierarchy.value = hierarchyData.sort((a: any, b: any) => a.email.localeCompare(b.email));
    }

    const chartRes = await fetch('/data/api/charts');
    if (chartRes.ok) {
      const resData = await chartRes.json();

      const inList = resData.inbound_rank || resData.inbounds || resData.user_rank || [];
      rawInboundData.value = inList.map((item: any) => ({
        value: item.count || item.Count || 1,
        name: item.src_ip || item.user || item.email || item.Email || '未知客户端'
      }));

      const outList = resData.outbound_rank || resData.outbounds || [];
      rawOutboundData.value = outList.map((item: any) => ({
        value: item.count || item.Count || 1,
        name: item.outbound || item.outbound_tag || item.Tag || '其他出口'
      }));
    }

    if (!selectedEmail.value && userHierarchy.value.length > 0) {
      const firstUser = userHierarchy.value[0];
      expandedUsers.value.add(firstUser.email);
      if (firstUser.ips && firstUser.ips.length > 0) {
        await selectEntity(firstUser.email, firstUser.ips[0]);
      }
    } else if (selectedEmail.value && selectedIP.value) {
      const res = await fetch(`/data/api/ip-targets?email=${encodeURIComponent(selectedEmail.value)}&ip=${encodeURIComponent(selectedIP.value)}`);
      if (res.ok) ipSpecificTargets.value = await res.json();
    }

    await nextTick();
    renderCharts();
  } catch (error) {
    console.error('核心数据链路更新故障:', error);
  }
};

const renderCharts = () => {
  const inDom = document.getElementById('inboundChart');
  if (inDom) {
    if (!inboundChart) inboundChart = echarts.init(inDom);

    let finalInboundData = [...rawInboundData.value];
    if (finalInboundData.length === 0 && userHierarchy.value.length > 0) {
      finalInboundData = userHierarchy.value.map(u => ({
        value: u.ips.length,
        name: u.email
      }));
    }

    inboundChart.setOption({
      backgroundColor: 'transparent',
      color: ['#3b82f6', '#10b981', '#f59e0b', '#8b5cf6', '#06b6d4', '#64748b'],
      tooltip: {
        trigger: 'item',
        backgroundColor: 'rgba(15, 23, 42, 0.95)',
        borderColor: 'rgba(255, 255, 255, 0.1)',
        borderWidth: 1,
        textStyle: { color: '#e2e8f0', fontSize: 11, fontFamily: 'monospace' },
        formatter: '{b}<br/>请求占比: <b>{c}</b> ({d}%)'
      },
      legend: { show: false },
      series: [{
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['50%', '50%'],
        minAngle: 15,
        avoidLabelOverlap: true,
        itemStyle: { borderRadius: 6, borderColor: '#0d111c', borderWidth: 2 },
        label: {
          show: true,
          position: 'outside',
          formatter: '{b}: {d}%',
          color: '#94a3b8',
          fontSize: 11,
          fontFamily: 'monospace'
        },
        labelLine: {
          show: true,
          length: 12,
          length2: 8,
          lineStyle: { color: 'rgba(255,255,255,0.15)' }
        },
        data: finalInboundData
      }]
    });
  }

  const outDom = document.getElementById('outboundChart');
  if (outDom) {
    if (!outboundChart) outboundChart = echarts.init(outDom);

    let finalOutboundData = [...rawOutboundData.value];
    if (finalOutboundData.length === 0) {
      finalOutboundData = [
        { value: overview.value.total_requests || 10, name: 'LISA_ISP_1' },
        { value: Math.floor((overview.value.total_requests || 10) * 0.2), name: 'SOCKS5_OUT' }
      ];
    }

    outboundChart.setOption({
      backgroundColor: 'transparent',
      color: ['#10b981', '#3b82f6', '#f59e0b', '#8b5cf6', '#06b6d4', '#64748b'],
      tooltip: {
        trigger: 'item',
        backgroundColor: 'rgba(15, 23, 42, 0.95)',
        borderColor: 'rgba(255, 255, 255, 0.1)',
        borderWidth: 1,
        textStyle: { color: '#e2e8f0', fontSize: 11, fontFamily: 'monospace' },
        formatter: '{b}<br/>路由计数: <b>{c}</b> ({d}%)'
      },
      legend: { show: false },
      series: [{
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['50%', '50%'],
        minAngle: 15,
        avoidLabelOverlap: true,
        itemStyle: { borderRadius: 6, borderColor: '#0d111c', borderWidth: 2 },
        label: {
          show: true,
          position: 'outside',
          formatter: '{b}: {d}%',
          color: '#94a3b8',
          fontSize: 11,
          fontFamily: 'monospace'
        },
        labelLine: {
          show: true,
          length: 12,
          length2: 8,
          lineStyle: { color: 'rgba(255,255,255,0.15)' }
        },
        data: finalOutboundData
      }]
    });
  }
};

onMounted(() => {
  fetchData();
  setInterval(fetchData, 5000);

  window.addEventListener('resize', () => {
    inboundChart?.resize();
    outboundChart?.resize();
  });
});
</script>

<style scoped>
/* ==========================================================================
   硬核暗黑科技原生态 CSS 样式面板 - 彻底告别全局样式死锁和黑字熔断灾难
   ========================================================================== */

.dashboard-root {
  box-sizing: border-box;
  min-height: 100vh;
  width: 100vw;
  background: radial-gradient(circle at 50% 0%, #11162d 0%, #070913 70%);
  padding: 32px 24px;
  color: #e2e8f0;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  overflow-x: hidden;
  text-align: left !important;
}

/* 顶部 Header 排版 */
.dashboard-header {
  max-w: 1280px;
  margin: 0 auto 32px auto;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}
@media (min-width: 640px) {
  .dashboard-header { flex-direction: row; align-items: center; }
}

.header-title h1 {
  font-size: 28px;
  font-weight: 700;
  color: #ffffff;
  margin: 0;
  letter-spacing: -0.5px;
}
.header-title p {
  font-size: 13px;
  color: #94a3b8;
  margin: 4px 0 0 0;
}

/* 运行状态呼吸灯 */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border-radius: 9999px;
  border: 1px solid rgba(16, 185, 129, 0.2);
  bg-color: rgba(16, 185, 129, 0.08);
  background: rgba(16, 185, 129, 0.06);
  padding: 6px 14px;
  font-size: 12px;
  font-weight: 600;
  color: #34d399;
  backdrop-filter: blur(8px);
}
.pulse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #10b981;
  box-shadow: 0 0 12px #10b981;
}

/* 核心网格布局布局 */
.dashboard-main {
  max-w: 1280px;
  margin: 0 auto;
}
.metrics-grid {
  display: grid;
  grid-template-cols: 1fr;
  gap: 24px;
  margin-bottom: 32px;
}
@media (min-width: 768px) {
  .metrics-grid { grid-template-cols: repeat(3, 1fr); }
}

/* 核心：仿 Flutter Card 立体晶格高拟真卡片 */
.flutter-card {
  position: relative;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(22, 28, 45, 0.6);
  padding: 24px;
  backdrop-filter: blur(16px);
  box-sizing: border-box;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.hover-lift:hover {
  transform: translateY(-5px);
}
.shadow-blue { box-shadow: 0 16px 36px rgba(0, 0, 0, 0.5), inset 0 1px 0 rgba(255,255,255,0.05); }
.shadow-blue:hover { border-color: rgba(59, 130, 246, 0.4); box-shadow: 0 20px 40px rgba(59, 130, 246, 0.15); }

.shadow-purple { box-shadow: 0 16px 36px rgba(0, 0, 0, 0.5), inset 0 1px 0 rgba(255,255,255,0.05); }
.shadow-purple:hover { border-color: rgba(168, 85, 247, 0.4); box-shadow: 0 20px 40px rgba(168, 85, 247, 0.15); }

.shadow-emerald { box-shadow: 0 16px 36px rgba(0, 0, 0, 0.5), inset 0 1px 0 rgba(255,255,255,0.05); }
.shadow-emerald:hover { border-color: rgba(16, 185, 129, 0.4); box-shadow: 0 20px 40px rgba(16, 185, 129, 0.15); }

/* 卡片内部指标文案样式 */
.card-label {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 1px;
  color: #94a3b8;
  margin-bottom: 12px;
}
.card-value {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 38px;
  font-weight: 700;
  letter-spacing: -1px;
}
.value-blue { color: #60a5fa; }
.value-purple { color: #c084fc; }
.value-emerald { color: #34d399; }

/* 图表区块 */
.charts-grid {
  display: grid;
  grid-template-cols: 1fr;
  gap: 24px;
  margin-bottom: 32px;
}
@media (min-width: 1024px) {
  .charts-grid { grid-template-cols: repeat(2, 1fr); }
}
.chart-container {
  height: 320px;
  width: 100%;
}

/* 装饰性小圆点标题 */
.section-sub-title {
  font-size: 14px;
  font-weight: 600;
  color: #f1f5f9;
  margin: 0 0 24px 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.dot { width: 6px; height: 6px; border-radius: 50%; }
.dot.blue { background-color: #3b82f6; box-shadow: 0 0 8px #3b82f6; }
.dot.emerald { background-color: #10b981; box-shadow: 0 0 8px #10b981; }

/* 下方级联分析面板结构 */
.analysis-section {
  margin-top: 32px;
}
.section-title-bar {
  border-left: 3px solid #3b82f6;
  padding-left: 16px;
  margin-bottom: 24px;
}
.section-title-bar h2 {
  font-size: 18px;
  font-weight: 700;
  color: #ffffff;
  margin: 0;
}
.section-title-bar p {
  font-size: 13px;
  color: #94a3b8;
  margin: 6px 0 0 0;
}

.split-panel {
  display: grid;
  grid-template-cols: 1fr;
  gap: 24px;
}
@media (min-width: 1024px) {
  .split-panel { grid-template-cols: 1fr 2fr; }
}

/* 面板头样式 */
.panel-header { margin-bottom: 20px; }
.panel-header h3 { font-size: 14px; font-weight: 700; color: #f1f5f9; margin: 0; }
.panel-header p { font-size: 12px; color: #64748b; margin: 4px 0 0 0; }

/* 侧边列表（用户实体目录） */
.list-container {
  max-height: 520px;
  overflow-y: auto;
  padding-right: 4px;
}
.user-group-item {
  border-radius: 10px;
  background: rgba(10, 15, 30, 0.4);
  border: 1px solid rgba(255, 255, 255, 0.03);
  margin-bottom: 10px;
  overflow: hidden;
}
.user-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.2s;
}
.user-row:hover { background: rgba(255, 255, 255, 0.04); }
.user-row.is-active { background: rgba(59, 130, 246, 0.12); color: #60a5fa; }

.email-text { font-family: monospace; font-weight: 500; word-break: break-all; }
.ip-count-badge {
  font-size: 11px;
  background: rgba(30, 41, 59, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.05);
  color: #94a3b8;
  padding: 2px 8px;
  border-radius: 6px;
}

/* 下拉 IP 部分 */
.ip-dropdown-box {
  background: rgba(5, 8, 16, 0.5);
  border-top: 1px solid rgba(255, 255, 255, 0.03);
  padding: 8px;
}
.ip-btn {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 6px;
  color: #94a3b8;
  font-family: monospace;
  font-size: 12px;
  padding: 8px 12px;
  text-align: left;
  cursor: pointer;
  box-sizing: border-box;
  margin-bottom: 4px;
  transition: all 0.2s;
}
.ip-btn:hover { background: rgba(255, 255, 255, 0.04); color: #f1f5f9; }
.ip-btn.is-selected {
  background: rgba(16, 185, 129, 0.1);
  border-color: rgba(16, 185, 129, 0.2);
  color: #34d399;
}
.active-dot { width: 6px; height: 6px; background: #10b981; border-radius: 50%; box-shadow: 0 0 6px #10b981; }

/* 右侧主面板：目标数据表格 */
.flex-between { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
.path-tags { display: flex; flex-wrap: wrap; items-center: center; gap: 6px; margin-top: 6px; font-size: 12px; }
.path-label { color: #64748b; }
.tag { font-family: monospace; font-size: 11px; padding: 2px 8px; border-radius: 4px; font-weight: 500; }
.tag-blue { background: rgba(59, 130, 246, 0.15); border: 1px solid rgba(59, 130, 246, 0.2); color: #60a5fa; }
.tag-emerald { background: rgba(16, 185, 129, 0.15); border: 1px solid rgba(16, 185, 129, 0.2); color: #34d399; }
.tag-none { font-style: italic; color: #475569; }

.top-badge { font-size: 11px; color: #64748b; background: rgba(255,255,255,0.03); border: 1px solid rgba(255,255,255,0.05); padding: 4px 10px; border-radius: 6px; }

/* 硬核暗色系数据表格架构 */
.table-wrapper {
  width: 100%;
  overflow-x: auto;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.04);
  background: rgba(5, 8, 16, 0.3);
}
.custom-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
  font-size: 13px;
}
.custom-table th {
  background: rgba(10, 15, 30, 0.8);
  color: #94a3b8;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: 12px 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}
.table-row {
  border-bottom: 1px solid rgba(255, 255, 255, 0.02);
  transition: background 0.15s;
}
.table-row:hover { background: rgba(255, 255, 255, 0.02); }

.custom-table td { padding: 12px 16px; color: #cbd5e1; font-family: monospace; }
.col-index { width: 70px; text-align: center; }
.cell-index { text-align: center; color: #475569; }
.col-count { width: 120px; text-align: right; }
.cell-count { text-align: right; font-weight: 600; }
.highlight-text { color: #34d399; }
.unit { font-size: 10px; font-weight: 400; color: #475569; margin-left: 2px; }

.cell-target { max-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.target-chip { background: rgba(0,0,0,0.2); border: 1px solid rgba(255,255,255,0.02); padding: 4px 8px; border-radius: 4px; color: #e2e8f0; }

/* 缺省状态 */
.empty-state, .table-empty { padding: 40px; text-align: center; color: #475569; font-style: italic; font-size: 13px; }

/* 深度图表穿透，防止渲染异常 */
:deep(.echarts) {
  background-color: transparent !important;
}
</style>
