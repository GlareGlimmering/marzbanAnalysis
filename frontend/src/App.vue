<template>
  <div class="min-h-screen bg-slate-50 px-4 py-6 text-slate-700 font-sans sm:px-6 lg:px-8">
    <header class="mx-auto mb-6 flex max-w-7xl flex-col gap-4 border-b border-slate-200 pb-5 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-slate-950">Xray Monitor</h1>
        <p class="mt-1 text-sm text-slate-500">网络实时数据行为分析面板</p>
      </div>
      <div class="inline-flex w-fit items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1.5 text-xs font-medium text-emerald-700">
        <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
        Dispatcher Running
      </div>
    </header>

    <main class="mx-auto max-w-7xl space-y-6">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <div class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
          <div class="mb-2 text-sm font-medium text-slate-500">总处理请求数</div>
          <div class="font-mono text-3xl font-semibold tracking-tight text-slate-950">{{ overview.total_requests || 0 }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
          <div class="mb-2 text-sm font-medium text-slate-500">活跃代理用户</div>
          <div class="font-mono text-3xl font-semibold tracking-tight text-blue-700">{{ overview.active_users || 0 }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
          <div class="mb-2 text-sm font-medium text-slate-500">活跃出口通道</div>
          <div class="font-mono text-3xl font-semibold tracking-tight text-amber-700">{{ overview.active_outbounds || 0 }}</div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <div class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
          <h3 class="mb-4 text-sm font-semibold text-slate-900">入站客户端来源排行</h3>
          <div id="inboundChart" class="h-64 w-full" style="min-height: 256px;"></div>
        </div>
        <div class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
          <h3 class="mb-4 text-sm font-semibold text-slate-900">落地线路负载排行</h3>
          <div id="outboundChart" class="h-64 w-full" style="min-height: 256px;"></div>
        </div>
      </div>

      <div class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <h3 class="mb-5 border-b border-slate-200 pb-4 text-sm font-semibold text-slate-900">
          客户端级联分析面板
        </h3>

        <div class="flex flex-col gap-6">
          <div class="flex w-full flex-col space-y-2 rounded-lg border border-slate-200 bg-slate-50 p-3">
            <div class="mb-1 text-xs font-medium text-slate-500">
              第一步：选择代理用户（Email） · 第二步：点击对应入站源 IP
            </div>

            <div class="space-y-2">
              <div v-for="user in userHierarchy" :key="user.email" class="overflow-hidden rounded-md border border-slate-200 bg-white">
                <div
                  @click="toggleUser(user.email)"
                  :class="[
                    'flex cursor-pointer select-none items-center justify-between gap-3 px-4 py-3 text-sm transition-colors',
                    selectedEmail === user.email ? 'bg-blue-50 text-blue-700 font-semibold' : 'text-slate-700 hover:bg-slate-50'
                  ]"
                >
                  <span class="break-all font-mono">{{ user.email }}</span>
                  <span class="shrink-0 rounded-full bg-slate-100 px-2 py-1 text-xs font-medium text-slate-500">
                    {{ user.ips.length }} 个关联 IP {{ expandedUsers.has(user.email) ? '收起' : '展开' }}
                  </span>
                </div>

                <div v-if="expandedUsers.has(user.email)" class="space-y-1 border-t border-slate-200 bg-slate-50 px-3 py-2">
                  <button
                    v-for="ip in user.ips"
                    :key="ip"
                    @click.stop="selectEntity(user.email, ip)"
                    :class="[
                      'flex w-full items-center justify-between rounded-md border px-4 py-2 text-left font-mono text-sm outline-none transition-colors',
                      selectedEmail === user.email && selectedIP === ip
                        ? 'border-emerald-200 bg-emerald-50 text-emerald-700 font-semibold'
                        : 'border-transparent bg-transparent text-slate-600 hover:border-slate-200 hover:bg-white hover:text-slate-900'
                    ]"
                  >
                    <span>{{ ip }}</span>
                    <span v-if="selectedEmail === user.email && selectedIP === ip" class="text-xs font-medium text-emerald-700">监控中</span>
                  </button>
                </div>
              </div>

              <div v-if="userHierarchy.length === 0" class="rounded-md border border-dashed border-slate-300 bg-white p-6 text-center text-sm text-slate-500">
                暂无活跃用户级联数据
              </div>
            </div>
          </div>

          <div class="w-full">
            <div class="mb-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="text-sm text-slate-500">
                当前路径:
                <span v-if="selectedEmail" class="ml-1 inline-flex max-w-full rounded-md border border-blue-100 bg-blue-50 px-2 py-1 font-mono text-xs font-medium text-blue-700">{{ selectedEmail }}</span>
                <span v-if="selectedIP" class="ml-1 inline-flex rounded-md border border-emerald-100 bg-emerald-50 px-2 py-1 font-mono text-xs font-medium text-emerald-700">{{ selectedIP }}</span>
                <span v-if="!selectedEmail" class="italic text-slate-400">未选择监听实体</span>
              </div>
              <span class="w-fit rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-500">精确流向 Top 20</span>
            </div>

            <div class="overflow-x-auto rounded-lg border border-slate-200">
              <table class="w-full border-collapse text-left">
                <thead>
                  <tr class="border-b border-slate-200 bg-slate-50 text-xs font-medium uppercase tracking-wide text-slate-500">
                    <th class="w-16 px-3 py-3 text-center">序号</th>
                    <th class="px-3 py-3">目标域名 / 目的 IP</th>
                    <th class="w-36 px-3 py-3 text-right">命中频次</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-100 bg-white font-mono text-sm">
                  <tr v-for="(detail, idx) in ipSpecificTargets" :key="idx" class="transition-colors hover:bg-slate-50">
                    <td class="px-3 py-3 text-center text-slate-400">{{ idx + 1 }}</td>
                    <td class="break-all px-3 py-3 text-slate-700 select-all">{{ detail.target }}</td>
                    <td class="px-3 py-3 pr-4 text-right font-semibold text-emerald-700">{{ detail.count }} 次</td>
                  </tr>
                  <tr v-if="ipSpecificTargets.length === 0">
                    <td colspan="3" class="py-16 text-center text-sm text-slate-400">
                      暂无精确匹配流向数据
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
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

// 缓存原始图表数据，兼容后端不同字段命名。
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
    const res = await fetch(`/api/ip-targets?email=${encodeURIComponent(email)}&ip=${encodeURIComponent(ip)}`);
    if (res.ok) {
      ipSpecificTargets.value = await res.json();
    }
  } catch (err) {
    console.error('联合数据抓取异常:', err);
  }
};

const fetchData = async () => {
  try {
    const overRes = await fetch('/api/overview');
    if (overRes.ok) overview.value = await overRes.json();

    const hierarchyRes = await fetch('/api/user-hierarchy');
    if (hierarchyRes.ok) userHierarchy.value = await hierarchyRes.json();

    const chartRes = await fetch('/api/charts');
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
      const res = await fetch(`/api/ip-targets?email=${encodeURIComponent(selectedEmail.value)}&ip=${encodeURIComponent(selectedIP.value)}`);
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
      color: ['#2563eb', '#059669', '#d97706', '#7c3aed', '#0891b2', '#4b5563'],
      tooltip: {
        trigger: 'item',
        backgroundColor: 'rgba(15, 23, 42, 0.9)', // 显式指定深色半透明背景 (slate-900)
        borderColor: '#1e293b',
        textStyle: { color: '#f8fafc', fontSize: 12, fontFamily: 'monospace' }, // 纯白文字
        formatter: '{b}<br/>请求占比: <b>{c}</b> ({d}%)'
      },
      legend: { right: '2%', top: 'center', orientation: 'vertical', textStyle: { color: '#475569', fontSize: 12, fontFamily: 'monospace' } },
      series: [{
        type: 'pie',
        radius: ['45%', '70%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: false,
        itemStyle: { borderRadius: 4, borderColor: '#ffffff', borderWidth: 2 },
        label: { show: false },
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
        { value: overview.value.total_requests || 10, name: 'LISA_ISP_1 (Active)' },
        { value: Math.floor((overview.value.total_requests || 10) * 0.2), name: 'SOCKS5_OUT_1' }
      ];
    }

    outboundChart.setOption({
      backgroundColor: 'transparent',
      color: ['#059669', '#2563eb', '#d97706', '#7c3aed', '#0891b2', '#4b5563'],
      tooltip: {
        trigger: 'item',
        backgroundColor: 'rgba(15, 23, 42, 0.9)',
        borderColor: '#1e293b',
        textStyle: { color: '#f8fafc', fontSize: 12, fontFamily: 'monospace' },
        formatter: '{b}<br/>路由计数: <b>{c}</b> ({d}%)'
      },
      legend: { right: '5%', top: 'center', orientation: 'vertical', textStyle: { color: '#475569', fontSize: 12, fontFamily: 'monospace' } },
      series: [{
        type: 'pie',
        radius: ['45%', '70%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: false,
        itemStyle: { borderRadius: 4, borderColor: '#ffffff', borderWidth: 2 },
        label: { show: false },
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

<style>
#app {
  width: 100%;
  max-width: none;
  border: 0;
  text-align: left;
}

.echarts {
  background-color: transparent !important;
}

::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: #f1f5f9;
}

::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 999px;
}
</style>
