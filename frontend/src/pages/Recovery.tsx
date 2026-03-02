import { useState } from 'react';
import { RotateCcw } from 'lucide-react';

export default function Recovery() {
  const [recoveryId, setRecoveryId] = useState('');

  const startRecovery = async () => {
    if (!recoveryId) return;
    try {
      const res = await fetch(`/api/recovery/start?backup_id=${recoveryId}`, { method: 'POST' });
      if (res.ok) {
        alert('Recovery started successfully');
        setRecoveryId('');
      } else {
        alert('Failed to start recovery');
      }
    } catch (err) {
      console.error(err);
      alert('Network error while starting recovery');
    }
  };

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden max-w-2xl">
        <div className="px-6 py-4 border-b border-gray-200 bg-gray-50">
          <h3 className="font-semibold">Start Recovery</h3>
        </div>
        <div className="p-6 space-y-4">
          <p className="text-sm text-gray-600">Enter a Backup ID to restore files from S3 to the original local path.</p>
          <div className="flex gap-4">
            <input 
              value={recoveryId} 
              onChange={(e) => setRecoveryId(e.target.value)} 
              placeholder="Backup ID" 
              className="flex-1 px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 outline-none" 
            />
            <button 
              onClick={startRecovery} 
              className="bg-green-600 hover:bg-green-700 text-white font-medium px-6 py-2 rounded-lg transition-colors flex items-center gap-2"
            >
              <RotateCcw className="w-4 h-4" /> Recover
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}