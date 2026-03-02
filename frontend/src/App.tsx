import { BrowserRouter as Router, Routes, Route, NavLink } from 'react-router-dom';
import { LayoutDashboard, Server, Eye, Database, RotateCcw, ShieldCheck, RefreshCw } from 'lucide-react';
import { useState } from 'react';
import Status from './pages/Status';
import Endpoints from './pages/Endpoints';
import WatchItems from './pages/WatchItems';
import Backups from './pages/Backups';
import Recovery from './pages/Recovery';

function App() {
  const [loading, setLoading] = useState(false);
  const [systemOnline, setSystemOnline] = useState(true);

  // You can use a context or global state to trigger refreshes across pages
  // For simplicity, we just have a dummy refresh state
  const [refreshKey, setRefreshKey] = useState(0);

  const refreshAll = () => {
    setLoading(true);
    setRefreshKey(prev => prev + 1);
    setTimeout(() => setLoading(false), 500); // Simulate refresh time
  };

  const menu = [
    { id: 'status', path: '/', label: 'Dashboard', icon: LayoutDashboard },
    { id: 'endpoints', path: '/endpoints', label: 'Endpoints', icon: Server },
    { id: 'watch', path: '/watch', label: 'Watch Items', icon: Eye },
    { id: 'backups', path: '/backups', label: 'Backups', icon: Database },
    { id: 'recovery', path: '/recovery', label: 'Recovery', icon: RotateCcw },
  ];

  return (
    <Router>
      <div className="min-h-screen flex bg-gray-50 text-gray-900 font-sans">
        {/* Sidebar */}
        <aside className="w-64 bg-slate-900 text-white flex-shrink-0 hidden md:flex flex-col">
          <div className="p-6 flex items-center gap-3">
            <ShieldCheck className="text-blue-400 w-6 h-6" />
            <h1 className="text-xl font-bold tracking-tight">S3 Backup</h1>
          </div>
          <nav className="flex-1 px-4 space-y-1">
            {menu.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.id}
                  to={item.path}
                  className={({ isActive }) =>
                    `w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors text-sm font-medium ${
                      isActive
                        ? 'bg-slate-800 text-white'
                        : 'text-slate-400 hover:bg-slate-800 hover:text-white'
                    }`
                  }
                >
                  <Icon className="w-4 h-4" />
                  <span>{item.label}</span>
                </NavLink>
              );
            })}
          </nav>
          <div className="p-4 border-t border-slate-800">
            <div className="flex items-center gap-3 px-3 py-2">
              <div className={`w-2 h-2 rounded-full ${systemOnline ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`}></div>
              <span className="text-xs text-slate-400 font-medium">
                {systemOnline ? 'System Online' : 'System Offline'}
              </span>
            </div>
          </div>
        </aside>

        {/* Main Content */}
        <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
          {/* Header */}
          <header className="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-8">
            <Routes>
              {menu.map((item) => (
                <Route key={item.id} path={item.path} element={<h2 className="text-lg font-semibold text-gray-800">{item.label}</h2>} />
              ))}
            </Routes>
            <div className="flex items-center gap-4">
              <button onClick={refreshAll} className="p-2 text-gray-500 hover:bg-gray-100 rounded-full transition-colors" title="Refresh All">
                <RefreshCw className={`w-5 h-5 ${loading ? 'animate-spin' : ''}`} />
              </button>
            </div>
          </header>

          {/* Content Area */}
          <div className="flex-1 overflow-y-auto p-8 relative">
            <Routes>
              <Route path="/" element={<Status refreshKey={refreshKey} setSystemOnline={setSystemOnline} />} />
              <Route path="/endpoints" element={<Endpoints refreshKey={refreshKey} />} />
              <Route path="/watch" element={<WatchItems refreshKey={refreshKey} />} />
              <Route path="/backups" element={<Backups refreshKey={refreshKey} />} />
              <Route path="/recovery" element={<Recovery />} />
            </Routes>
          </div>
        </main>
      </div>
    </Router>
  );
}

export default App;
