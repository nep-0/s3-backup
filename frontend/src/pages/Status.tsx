import { useState, useEffect } from 'react';
import { Server, Eye, Database } from 'lucide-react';

interface StatusData {
  endpoints?: any[];
  watch_items?: any[];
  backups?: any[];
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