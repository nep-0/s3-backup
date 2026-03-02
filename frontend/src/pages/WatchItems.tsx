import { useState, useEffect } from 'react';
import { Plus, Trash2, Play } from 'lucide-react';

interface WatchItem {
  id?: number;
  path: string;
  target_path: string;
  excludes: string[];
  endpoint_id: number;
  enabled: boolean;
}

interface Endpoint {
  id: number;
  name: string;
  endpoint: string;
}

export default function WatchItems({ refreshKey }: { refreshKey: number }) {
  const [watchItems, setWatchItems] = useState<WatchItem[]>([]);
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [newWatch, setNewWatch] = useState({
    path: '', target_path: '', endpoint_id: '', excludes: '', enabled: true
  });

  const loadWatch = () => {
    fetch('/api/watch')
      .then(res => res.json())
      .then(data => setWatchItems(data || []))
      .catch(console.error);
  };

  const loadEndpoints = () => {
    fetch('/api/endpoints')
      .then(res => res.json())
      .then(data => setEndpoints(data || []))
      .catch(console.error);
  };

  useEffect(() => {
    loadWatch();
    loadEndpoints();
  }, [refreshKey]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    const checked = (e.target as HTMLInputElement).checked;
    setNewWatch(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };

  const createWatch = async () => {
    try {
      const payload = {
        ...newWatch,
        endpoint_id: parseInt(newWatch.endpoint_id, 10),
        excludes: newWatch.excludes.split(',').map(v => v.trim()).filter(Boolean)
      };
      const res = await fetch('/api/watch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (res.ok) {
        setNewWatch({ path: '', target_path: '', endpoint_id: '', excludes: '', enabled: true });
        loadWatch();
      } else {
        alert('Error creating watch item');
      }
    } catch (err) {
      console.error(err);
    }
  };

  const deleteWatch = async (id: number) => {
    if (!confirm('Are you sure?')) return;
    try {
      const res = await fetch(`/api/watch?id=${id}`, { method: 'DELETE' });
      if (res.ok) {
        loadWatch();
      }
    } catch (err) {
      console.error(err);
    }
  };

  const triggerBackup = async (id: number) => {
    try {
      const res = await fetch(`/api/backup/trigger?watch_id=${id}`, { method: 'POST' });
      if (res.ok) {
        alert('Backup triggered successfully');
      } else {
        alert('Error triggering backup');
      }
    } catch (err) {
      console.error(err);
    }
  };

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 bg-gray-50">
          <h3 className="font-semibold">Add New Watch Item</h3>
        </div>
        <div className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <input name="path" value={newWatch.path} onChange={handleInputChange} placeholder="Local Path to watch" className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" />
            <input name="target_path" value={newWatch.target_path} onChange={handleInputChange} placeholder="Target S3 Path" className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" />
            <input name="excludes" value={newWatch.excludes} onChange={handleInputChange} placeholder="Excludes (comma separated)" className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" />
            <select name="endpoint_id" value={newWatch.endpoint_id} onChange={handleInputChange} className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none bg-white">
              <option value="">Select Endpoint</option>
              {endpoints.map(ep => (
                <option key={ep.id} value={ep.id}>{`${ep.name} (${ep.endpoint})`}</option>
              ))}
            </select>
            <div className="flex items-center gap-4 px-2">
              <label className="flex items-center gap-2 cursor-pointer">
                <input name="enabled" type="checkbox" checked={newWatch.enabled} onChange={handleInputChange} className="rounded text-blue-600" />
                <span className="text-sm">Enabled</span>
              </label>
            </div>
            <button onClick={createWatch} className="md:col-span-2 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 rounded-lg transition-colors flex items-center justify-center gap-2">
              <Plus className="w-4 h-4" /> Add Watch
            </button>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-200">
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">ID</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Path</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Target</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {watchItems.map(w => (
              <tr key={w.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-6 py-4 text-sm text-gray-500">{w.id}</td>
                <td className="px-6 py-4 text-sm font-medium">{w.path}</td>
                <td className="px-6 py-4 text-sm text-gray-600">{w.target_path}</td>
                <td className="px-6 py-4 text-sm">
                  <span className={`px-2 py-1 rounded-full text-xs font-bold ${w.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-700'}`}>
                    {w.enabled ? 'Active' : 'Disabled'}
                  </span>
                </td>
                <td className="px-6 py-4 text-right space-x-2">
                  <button onClick={() => w.id && triggerBackup(w.id)} className="text-blue-600 hover:text-blue-800 p-1" title="Trigger Backup">
                    <Play className="w-4 h-4" />
                  </button>
                  <button onClick={() => w.id && deleteWatch(w.id)} className="text-red-600 hover:text-red-800 p-1" title="Delete">
                    <Trash2 className="w-4 h-4" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}