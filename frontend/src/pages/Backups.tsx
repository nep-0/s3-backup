import { useState, useEffect } from 'react';
import { Info, X, RotateCcw } from 'lucide-react';

interface Backup {
  id: number;
  watch_item_id: number;
  endpoint_id: number;
  watch_path: string;
  target_path: string;
  started_at: string;
  completed_at: string;
  status: string;
  total_files: number;
  total_bytes: number;
  error: string;
}

export default function Backups({ refreshKey }: { refreshKey: number }) {
  const [backups, setBackups] = useState<Backup[]>([]);
  const [selectedBackup, setSelectedBackup] = useState<any>(null);
  const [recoveringId, setRecoveringId] = useState<number | null>(null);

  const loadBackups = () => {
    fetch('/api/backups/list')
      .then(res => res.json())
      .then(data => setBackups(data || []))
      .catch(console.error);
  };

  useEffect(() => {
    loadBackups();
  }, [refreshKey]);

  const loadBackupDetail = async (id: number) => {
    try {
      const res = await fetch(`/api/backups/detail?id=${id}`);
      if (res.ok) {
        const data = await res.json();
        setSelectedBackup(data);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const startRecovery = async (id: number) => {
    if (!confirm(`Start recovery for backup ${id}?`)) return;
    try {
      setRecoveringId(id);
      const res = await fetch(`/api/recovery/start?backup_id=${id}`, { method: 'POST' });
      if (res.ok) {
        alert('Recovery started successfully');
      } else {
        alert('Failed to start recovery');
      }
    } catch (err) {
      console.error(err);
      alert('Network error while starting recovery');
    } finally {
      setRecoveringId(null);
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-200">
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">ID</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Watch Path</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Target</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Files</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Size</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {backups.map(b => (
              <tr key={b.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-6 py-4 text-sm text-gray-500">{b.id}</td>
                <td className="px-6 py-4 text-sm font-medium max-w-[260px] truncate" title={b.watch_path}>{b.watch_path}</td>
                <td className="px-6 py-4 text-sm text-gray-600 max-w-[220px] truncate" title={b.target_path}>{b.target_path}</td>
                <td className="px-6 py-4 text-sm">
                  <span className={`px-2 py-1 rounded-full text-xs font-bold ${
                    b.status === 'completed' ? 'bg-green-100 text-green-700' :
                    b.status === 'running' ? 'bg-blue-100 text-blue-700' :
                    'bg-red-100 text-red-700'
                  }`}>
                    {b.status}
                  </span>
                </td>
                <td className="px-6 py-4 text-sm text-gray-600">{b.total_files}</td>
                <td className="px-6 py-4 text-sm text-gray-600">{formatBytes(b.total_bytes)}</td>
                <td className="px-6 py-4 text-right space-x-2">
                  <button onClick={() => startRecovery(b.id)} className="text-green-600 hover:text-green-800 p-1" title="Recover">
                    <RotateCcw className={`w-4 h-4 ${recoveringId === b.id ? 'animate-spin' : ''}`} />
                  </button>
                  <button onClick={() => loadBackupDetail(b.id)} className="text-blue-600 hover:text-blue-800 p-1" title="Details">
                    <Info className="w-4 h-4" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Modal for Details */}
      {selectedBackup && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-xl shadow-xl w-full max-w-3xl flex flex-col max-h-[80vh]">
            <div className="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
              <h3 className="font-semibold">Backup Details</h3>
              <button onClick={() => setSelectedBackup(null)} className="text-gray-500 hover:text-gray-700">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="p-6 overflow-y-auto">
              <pre className="bg-gray-900 text-green-400 p-4 rounded-lg text-xs overflow-x-auto whitespace-pre-wrap">
                {JSON.stringify(selectedBackup, null, 2)}
              </pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}