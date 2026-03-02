import { useState, useEffect } from 'react';
import { Plus, Trash2 } from 'lucide-react';

interface Endpoint {
  id?: number;
  name: string;
  endpoint: string;
  access_key: string;
  secret_key: string;
  bucket: string;
  prefix: string;
  region: string;
  use_ssl: boolean;
  path_style: boolean;
}

export default function Endpoints({ refreshKey }: { refreshKey: number }) {
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [newEndpoint, setNewEndpoint] = useState<Endpoint>({
    name: '', endpoint: '', access_key: '', secret_key: '',
    bucket: '', prefix: '', region: '', use_ssl: false, path_style: true
  });

  const loadEndpoints = () => {
    fetch('/api/endpoints')
      .then(res => res.json())
      .then(data => setEndpoints(data || []))
      .catch(console.error);
  };

  useEffect(() => {
    loadEndpoints();
  }, [refreshKey]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value, type, checked } = e.target;
    setNewEndpoint(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };

  const createEndpoint = async () => {
    try {
      const res = await fetch('/api/endpoints', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newEndpoint)
      });
      if (res.ok) {
        setNewEndpoint({ name: '', endpoint: '', access_key: '', secret_key: '', bucket: '', prefix: '', region: '', use_ssl: false, path_style: true });
        loadEndpoints();
      } else {
        alert('Error creating endpoint');
      }
    } catch (err) {
      console.error(err);
    }
  };

  const deleteEndpoint = async (id: number) => {
    if (!confirm('Are you sure?')) return;
    try {
      const res = await fetch(`/api/endpoints?id=${id}`, { method: 'DELETE' });
      if (res.ok) {
        loadEndpoints();
      }
    } catch (err) {
      console.error(err);
    }
  };

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 bg-gray-50">
          <h3 className="font-semibold">Add New Endpoint</h3>
        </div>
        <div className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <input name="name" value={newEndpoint.name} onChange={handleInputChange} placeholder="Name (e.g. AWS S3)" className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" />
            <input name="endpoint" value={newEndpoint.endpoint} onChange={handleInputChange} placeholder="Endpoint (host:port)" className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" />
            <input name="access_key" value={newEndpoint.access_key} onChange={handleInputChange} placeholder="Access Key" className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" />
            <input name="secret_key" type="password" value={newEndpoint.secret_key} onChange={handleInputChange} placeholder="Secret Key" className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" />
            <input name="bucket" value={newEndpoint.bucket} onChange={handleInputChange} placeholder="Bucket Name" className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" />
            <input name="prefix" value={newEndpoint.prefix} onChange={handleInputChange} placeholder="Prefix (optional)" className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" />
            <input name="region" value={newEndpoint.region} onChange={handleInputChange} placeholder="Region (e.g. us-east-1)" className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" />
            <div className="flex items-center gap-4 px-2">
              <label className="flex items-center gap-2 cursor-pointer">
                <input name="use_ssl" type="checkbox" checked={newEndpoint.use_ssl} onChange={handleInputChange} className="rounded text-blue-600" />
                <span className="text-sm">SSL</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input name="path_style" type="checkbox" checked={newEndpoint.path_style} onChange={handleInputChange} className="rounded text-blue-600" />
                <span className="text-sm">Path Style</span>
              </label>
            </div>
            <button onClick={createEndpoint} className="md:col-span-3 bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 rounded-lg transition-colors flex items-center justify-center gap-2">
              <Plus className="w-4 h-4" /> Add Endpoint
            </button>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-200">
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">ID</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Name</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Endpoint</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Bucket</th>
              <th className="px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {endpoints.map(ep => (
              <tr key={ep.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-6 py-4 text-sm text-gray-500">{ep.id}</td>
                <td className="px-6 py-4 text-sm font-medium">{ep.name}</td>
                <td className="px-6 py-4 text-sm text-gray-600">{ep.endpoint}</td>
                <td className="px-6 py-4 text-sm text-gray-600">{ep.bucket}</td>
                <td className="px-6 py-4 text-right">
                  <button onClick={() => ep.id && deleteEndpoint(ep.id)} className="text-red-600 hover:text-red-800 p-1">
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