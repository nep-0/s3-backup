import { useState, useEffect } from 'react';
import { Server, Eye, Database, BarChart3 } from 'lucide-react';

interface StorageTotals {
  original_bytes: number;
  compressed_bytes: number;
}

interface StorageTrendPoint {
  backup_id: number;
  timestamp: string;
  original_bytes: number;
  compressed_bytes: number;
}

interface StatusData {
  endpoints?: any[];
  watch_items?: any[];
  backups?: any[];
  storage_totals?: StorageTotals;
  storage_trend?: StorageTrendPoint[];
}

export default function Status({ refreshKey, setSystemOnline }: { refreshKey: number, setSystemOnline: (status: boolean) => void }) {
  const [status, setStatus] = useState<StatusData>({});

  useEffect(() => {
    fetch('/api/status')
      .then((res) => {
        if (!res.ok) throw new Error('Network error');
        return res.json();
      })
      .then((data) => {
        setStatus(data || {});
        setSystemOnline(true);
      })
      .catch((err) => {
        console.error(err);
        setSystemOnline(false);
      });
  }, [refreshKey, setSystemOnline]);

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const trend = (status.storage_trend || []).slice().reverse();
  const maxOriginal = Math.max(1, ...trend.map((t) => t.original_bytes || 0));
  const maxCompressed = Math.max(1, ...trend.map((t) => t.compressed_bytes || 0));
  const maxValue = Math.max(maxOriginal, maxCompressed, 1);
  const chartHeight = 180;
  const chartWidth = 600;
  const padding = 30;

  const toX = (index: number) => {
    if (trend.length <= 1) return padding;
    return padding + (index / (trend.length - 1)) * (chartWidth - padding * 2);
  };

  const toY = (value: number) => {
    const ratio = value / maxValue;
    return chartHeight - padding - ratio * (chartHeight - padding * 2);
  };

  const buildPath = (key: 'original_bytes' | 'compressed_bytes') => {
    return trend
      .map((point, index) => `${index === 0 ? 'M' : 'L'} ${toX(index)} ${toY(point[key] || 0)}`)
      .join(' ');
  };

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white p-6 rounded-xl border border-gray-200 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <span className="text-sm font-medium text-gray-500 uppercase tracking-wider">Endpoints</span>
            <Server className="w-5 h-5 text-blue-500" />
          </div>
          <div className="text-3xl font-bold">{status.endpoints?.length || 0}</div>
        </div>
        <div className="bg-white p-6 rounded-xl border border-gray-200 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <span className="text-sm font-medium text-gray-500 uppercase tracking-wider">Watch Items</span>
            <Eye className="w-5 h-5 text-purple-500" />
          </div>
          <div className="text-3xl font-bold">{status.watch_items?.length || 0}</div>
        </div>
        <div className="bg-white p-6 rounded-xl border border-gray-200 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <span className="text-sm font-medium text-gray-500 uppercase tracking-wider">Total Backups</span>
            <Database className="w-5 h-5 text-green-500" />
          </div>
          <div className="text-3xl font-bold">{status.backups?.length || 0}</div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="bg-white p-6 rounded-xl border border-gray-200 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <span className="text-sm font-medium text-gray-500 uppercase tracking-wider">Original Storage</span>
            <Database className="w-5 h-5 text-indigo-500" />
          </div>
          <div className="text-2xl font-bold">{formatBytes(status.storage_totals?.original_bytes || 0)}</div>
          <p className="text-xs text-gray-500 mt-1">Raw data size</p>
        </div>
        <div className="bg-white p-6 rounded-xl border border-gray-200 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <span className="text-sm font-medium text-gray-500 uppercase tracking-wider">Compressed Storage</span>
            <Database className="w-5 h-5 text-teal-500" />
          </div>
          <div className="text-2xl font-bold">{formatBytes(status.storage_totals?.compressed_bytes || 0)}</div>
          <p className="text-xs text-gray-500 mt-1">Stored in S3 (Zstd)</p>
        </div>
        <div className="bg-white p-6 rounded-xl border border-gray-200 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <span className="text-sm font-medium text-gray-500 uppercase tracking-wider">Compression Ratio</span>
            <BarChart3 className="w-5 h-5 text-amber-500" />
          </div>
          <div className="text-2xl font-bold">
            {status.storage_totals?.original_bytes
              ? `${((status.storage_totals.compressed_bytes / status.storage_totals.original_bytes) * 100).toFixed(1)}%`
              : '0%'}
          </div>
          <p className="text-xs text-gray-500 mt-1">Compressed / Original</p>
        </div>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 bg-gray-50 flex items-center justify-between">
          <h3 className="font-semibold">Storage Trend</h3>
          <span className="text-xs text-gray-500">Last {trend.length} backups</span>
        </div>
        <div className="p-6">
          {trend.length === 0 ? (
            <div className="text-sm text-gray-500">No backup data available yet.</div>
          ) : (
            <div className="w-full overflow-x-auto">
              <svg width={chartWidth} height={chartHeight} className="w-full max-w-full">
                <path d={buildPath('original_bytes')} fill="none" stroke="#6366f1" strokeWidth="2" />
                <path d={buildPath('compressed_bytes')} fill="none" stroke="#14b8a6" strokeWidth="2" />
              </svg>
              <div className="flex flex-wrap gap-4 text-xs text-gray-500 mt-4">
                <div className="flex items-center gap-2">
                  <span className="inline-block w-3 h-3 rounded-full bg-indigo-500"></span>
                  Original size
                </div>
                <div className="flex items-center gap-2">
                  <span className="inline-block w-3 h-3 rounded-full bg-teal-500"></span>
                  Compressed size
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 bg-gray-50 flex justify-between items-center">
          <h3 className="font-semibold">System Status (Raw)</h3>
        </div>
        <div className="p-6">
          <pre className="bg-gray-900 text-green-400 p-4 rounded-lg text-xs overflow-x-auto">
            {JSON.stringify(status, null, 2)}
          </pre>
        </div>
      </div>
    </div>
  );
}
