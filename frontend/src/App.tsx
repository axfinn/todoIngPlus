import React, { useState, useEffect, useRef } from 'react';
import { Routes, Route, Link } from 'react-router-dom';
import { useDispatch, useSelector } from 'react-redux';
import { useTranslation } from 'react-i18next';
import type { RootState, AppDispatch } from './app/store';
import { logout } from './features/auth/authSlice';
import useNotificationStream from './hooks/useNotificationStream';

import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import DashboardPage from './pages/DashboardPage';
import ReportsPage from './pages/ReportsPage';
import EventsPage from './pages/EventsPage';
import EventDetailPage from './pages/EventDetailPage';
import RemindersPage from './pages/RemindersPage';
import UnifiedBoardPage from './pages/UnifiedBoardPage';
import ProtectedRoute from './components/ProtectedRoute';

// 全局 now context: 统一 1s tick，避免多组件各自 setInterval 造成过度重绘（配合 backdrop-filter 会闪烁）
export const NowContext = React.createContext<number>(Date.now());
const NowProvider: React.FC<{children: React.ReactNode}> = ({children}) => {
  const [now, setNow] = useState(Date.now());
  const rafRef = useRef<number>();
  useEffect(()=>{
    let last = performance.now();
    const loop = (ts: number) => {
      if (ts - last >= 1000) { last = ts; setNow(Date.now()); }
      rafRef.current = requestAnimationFrame(loop);
    };
    rafRef.current = requestAnimationFrame(loop);
    return ()=> { if (rafRef.current) cancelAnimationFrame(rafRef.current); };
  },[]);
  return <NowContext.Provider value={now}>{children}</NowContext.Provider>;
};

const App: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const { isAuthenticated } = useSelector((state: RootState) => state.auth);
  const { t, i18n } = useTranslation();
  const [currentLanguage, setCurrentLanguage] = useState('en');
  const [githubStats, setGithubStats] = useState({ stars: 0, forks: 0 });
  const [isGitHubLoading, setIsGitHubLoading] = useState(false);
  const [githubError, setGithubError] = useState(false);
  // 背景图（纯展示用，不影响交互，放置在模糊层）
  const [bgUrl, setBgUrl] = useState<string>('');
  useEffect(()=> {
    const pool = [
      'https://picsum.photos/1600/900?random=11',
      'https://picsum.photos/1600/900?random=12',
      'https://picsum.photos/1600/900?random=13'
    ];
    const url = pool[Math.floor(Math.random()*pool.length)];
    const img = new Image();
    img.onload = () => setBgUrl(url);
    img.src = url;
    console.log('[BG] loading', url);
  }, []);

  // 将背景注入到 body::before
  useEffect(()=> {
    if (!bgUrl) return;
    document.body.classList.add('has-bg');
    document.body.style.setProperty('--app-bg-image', `url(${bgUrl})`);
    requestAnimationFrame(()=> document.body.classList.add('bg-visible'));
    // 诊断：下一帧读取计算样式确认 ::before 是否应用
    setTimeout(()=> {
      try {
        const beforeStyle = (getComputedStyle(document.body, '::before') as any).backgroundImage;
        if (!beforeStyle || beforeStyle === 'none') {
          console.warn('[BG] body::before 未检测到背景，启用 fallback 层');
          const fb = document.getElementById('app-bg-fallback');
          if (fb) fb.style.opacity = '1';
        } else {
          console.log('[BG] body::before 检测到背景:', beforeStyle);
        }
      } catch (e) {
        console.warn('[BG] 检测 body::before 背景异常', e);
      }
    }, 100);
    return () => {
      document.body.classList.remove('has-bg','bg-visible');
      document.body.style.removeProperty('--app-bg-image');
    };
  }, [bgUrl]);

  // 调试按键：b 切换卡片透明度，便于查看背景
  useEffect(()=> {
    const keyHandler = (e: KeyboardEvent) => {
      if (e.key.toLowerCase() === 'b' && (e.metaKey || e.ctrlKey)) {
        document.body.classList.toggle('debug-bg-transparent');
      }
    };
    window.addEventListener('keydown', keyHandler);
    return ()=> window.removeEventListener('keydown', keyHandler);
  }, []);

  useEffect(() => { setCurrentLanguage(i18n.language); }, [i18n.language]);

  // 开发调试：捕获全局点击与可能的遮挡元素
  useEffect(() => {
    if (process.env.NODE_ENV !== 'development') return;
    const handler = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      const info = target ? `${target.tagName}.${[...target.classList].join('.')}` : 'null';
      const pe = getComputedStyle(target).pointerEvents;
      const statusEl = document.getElementById('debug-pointer-status');
      if (statusEl) statusEl.textContent = `Click -> ${info} pe=${pe}`;
      // 检测顶部覆盖元素（同一点再探测）
      const el2 = document.elementFromPoint(e.clientX, e.clientY);
      if (el2 && el2 !== target && statusEl) {
        statusEl.textContent += ` topEl=${el2.tagName}.${[...el2.classList].join('.')}`;
      }
    };
    window.addEventListener('click', handler, { capture: true });
    return () => window.removeEventListener('click', handler, { capture: true } as any);
  }, []);

  useEffect(() => {
    // 获取GitHub项目统计信息
    setIsGitHubLoading(true);
    setGithubError(false);
    
    fetch('https://api.github.com/repos/axfinn/todoIng')
      .then(response => {
        if (!response.ok) {
          throw new Error('Network response was not ok');
        }
        return response.json();
      })
      .then(data => {
        setGithubStats({
          stars: data.stargazers_count || 0,
          forks: data.forks_count || 0
        });
        setIsGitHubLoading(false);
      })
      .catch(error => {
        console.error('Failed to fetch GitHub stats:', error);
        setIsGitHubLoading(false);
        setGithubError(true);
      });
  }, []);

  const handleLogout = () => {
    dispatch(logout());
  };

  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
    setCurrentLanguage(lng);
  };

  // 将 body class 与开关同步（面板蒙版快速关闭）
  // 同步透明度到 CSS 变量
  // 透明度控制已移除

  // 订阅通知（登录后）
  useNotificationStream(isAuthenticated);
  const unread = useSelector((s: RootState)=> s.notifications?.unread || 0);
  // 背景模糊强度控制
  const [bgBlur, setBgBlur] = useState<number>(()=> {
    const v = localStorage.getItem('ui.bgBlur');
    const n = v? parseInt(v,10): 20;
    return isNaN(n)? 20: Math.min(60, Math.max(4, n));
  });
  useEffect(()=> {
    document.body.style.setProperty('--app-bg-blur', bgBlur+'px');
    document.body.style.setProperty('--app-bg-blur-dark', Math.round(bgBlur*1.1)+'px');
    localStorage.setItem('ui.bgBlur', bgBlur.toString());
  }, [bgBlur]);

  return (
    <NowProvider>
  <div className="app-root d-flex flex-column min-vh-100 position-relative">
      {/* Fallback 背景层：当 body::before 不生效时仍可看到背景图 */}
      <div
        id="app-bg-fallback"
        className="app-bg-fallback"
        style={{
          backgroundImage: bgUrl ? `url(${bgUrl})` : undefined,
          opacity: bgUrl ? 1 : 0,
          transition: 'opacity .7s ease'
        }}
      />
  {/* 背景通过 body::before 注入，无需额外 DOM */}
      {/* Debug overlay for pointer interception (可随时移除) */}
      {process.env.NODE_ENV === 'development' && (
        <div style={{position:'fixed',bottom:4,right:4,zIndex:3000,fontSize:11,background:'rgba(0,0,0,0.4)',color:'#fff',padding:'2px 6px',borderRadius:4}}>
          <span id="debug-pointer-status">Ready</span>
        </div>
      )}
  {/* 背景已简化为纯色，去除 overlay */}
      <header>
        <nav className="navbar navbar-expand-lg navbar-dark bg-primary shadow-sm">
          <div className="container">
            <Link className="navbar-brand fw-bold d-flex align-items-center" to="/">
              <i className="bi bi-check2-circle me-2"></i>
              todoIng
            </Link>
            <button
              className="navbar-toggler"
              type="button"
              data-bs-toggle="collapse"
              data-bs-target="#navbarNav"
              aria-controls="navbarNav"
              aria-expanded="false"
              aria-label="Toggle navigation"
            >
              <span className="navbar-toggler-icon"></span>
            </button>
            <div className="collapse navbar-collapse" id="navbarNav">
              <ul className="navbar-nav me-auto mb-2 mb-lg-0">
                <li className="nav-item">
                  <Link className="nav-link" to="/">
                    {t('nav.home')}
                  </Link>
                </li>
                {isAuthenticated && (
                  <li className="nav-item">
                    <Link className="nav-link" to="/dashboard">
                      {t('nav.dashboard')}
                    </Link>
                  </li>
                )}
                {isAuthenticated && (
                  <li className="nav-item">
                    <Link className="nav-link" to="/events">
                      <i className="bi bi-calendar-event me-1"></i>
                      {t('nav.events')}
                    </Link>
                  </li>
                )}
        {isAuthenticated && (
                  <li className="nav-item">
                    <Link className="nav-link" to="/reminders">
                      <i className="bi bi-bell me-1"></i>
                      {t('nav.reminders')}
          {unread>0 && <span className="badge bg-danger ms-1" style={{fontSize:'0.65rem'}}>{unread}</span>}
                    </Link>
                  </li>
                )}
                {isAuthenticated && (
                  <li className="nav-item">
                    <Link className="nav-link" to="/reports">
                      {t('nav.reports')}
                    </Link>
                  </li>
                )}
                {isAuthenticated && (
                  <li className="nav-item">
                    <Link className="nav-link" to="/unified">
                      <i className="bi bi-collection me-1"></i>
                      {t('nav.unified')}
                    </Link>
                  </li>
                )}
              </ul>
              <ul className="navbar-nav mb-2 mb-lg-0 align-items-lg-center">
                {bgUrl && (
                  <li className="nav-item me-3">
                    <div className="dropdown">
                      <button className="btn btn-outline-light btn-sm dropdown-toggle" data-bs-toggle="dropdown">
                        <i className="bi bi-image"/> 背景
                      </button>
                      <div className="dropdown-menu dropdown-menu-end p-3 small" style={{minWidth:240}}>
                        <label className="form-label d-flex justify-content-between mb-1">
                          <span>模糊</span><span className="badge bg-secondary">{bgBlur}px</span>
                        </label>
                        <input type="range" min={4} max={60} step={1} value={bgBlur} className="form-range" onChange={e=> setBgBlur(parseInt(e.target.value,10))} />
                        <div className="d-flex gap-2 flex-wrap mt-2">
                          <button type="button" className="btn btn-sm btn-outline-secondary" onClick={()=> setBgBlur(10)}>浅</button>
                          <button type="button" className="btn btn-sm btn-outline-secondary" onClick={()=> setBgBlur(20)}>默认</button>
                          <button type="button" className="btn btn-sm btn-outline-secondary" onClick={()=> setBgBlur(32)}>重</button>
                          <button type="button" className="btn btn-sm btn-outline-secondary" onClick={()=> setBgBlur(45)}>更重</button>
                        </div>
                      </div>
                    </div>
                  </li>
                )}
                {/* 透明度调节面板已移除 */}
                <li className="nav-item dropdown">
                  <a className="btn btn-outline-light dropdown-toggle me-2" href="#" role="button" data-bs-toggle="dropdown" aria-expanded="false">
                    {currentLanguage === 'en' ? 'English' : '中文'}
                  </a>
                  <ul className="dropdown-menu">
                    <li><a className="dropdown-item" href="#" onClick={(e) => { e.preventDefault(); changeLanguage('en'); }}>English</a></li>
                    <li><hr className="dropdown-divider" /></li>
                    <li><a className="dropdown-item" href="#" onClick={(e) => { e.preventDefault(); changeLanguage('zh'); }}>中文</a></li>
                  </ul>
                </li>
                {isAuthenticated ? (
                  <li className="nav-item">
                    <button className="btn btn-outline-light" onClick={handleLogout}>
                      <i className="bi bi-box-arrow-right me-1"></i>
                      {t('nav.logout')}
                    </button>
                  </li>
                ) : (
                  <>
                    <li className="nav-item">
                      <Link className="nav-link" to="/register">
                        {t('nav.register')}
                      </Link>
                    </li>
                    <li className="nav-item">
                      <Link className="nav-link" to="/login">
                        {t('nav.login')}
                      </Link>
                    </li>
                  </>
                )}
              </ul>
            </div>
          </div>
        </nav>
      </header>
  <main className="flex-grow-1 app-content-wrapper">
        <Routes>
          <Route path="/" element={
            <div className="container py-5">
              <div className="row justify-content-center">
                <div className="col-md-8 text-center" style={{ 
                  backgroundColor: 'rgba(255, 255, 255, 0.8)',
                  borderRadius: '10px',
                  padding: '2rem',
                  boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)'
                }}>
                  <h1 className="display-4 fw-bold mb-4">{t('home.title')}</h1>
                  <p className="lead mb-4">
                    {t('home.description')}
                  </p>
                  {!isAuthenticated && (
                    <div className="d-grid gap-3 d-sm-flex justify-content-sm-center">
                      <Link to="/register" className="btn btn-primary btn-lg px-4 gap-3">
                        {t('home.getStarted')}
                      </Link>
                      <Link to="/login" className="btn btn-outline-primary btn-lg px-4">
                        {t('home.login')}
                      </Link>
                    </div>
                  )}
                  {isAuthenticated && (
                    <div className="d-grid gap-3 d-sm-flex justify-content-sm-center">
                      <Link to="/dashboard" className="btn btn-primary btn-lg px-4 gap-3">
                        {t('nav.dashboard')}
                      </Link>
                    </div>
                  )}
                  
                  <div className="mt-5">
                    <div className="d-flex align-items-center justify-content-center mb-3">
                      <i className="bi bi-github me-2"></i>
                      <a href="https://github.com/axfinn/todoIngPlus" target="_blank" rel="noopener noreferrer" className="text-decoration-none">
                        {t('github.fork')}
                      </a>
                    </div>
                    {isGitHubLoading && (
                      <div className="text-muted small">{t('github.loading')}</div>
                    )}
                    {githubError && !isGitHubLoading && (
                      <div className="text-danger small">{t('github.error')}</div>
                    )}
                    {!isGitHubLoading && !githubError && (
                      <div className="d-flex justify-content-center gap-3">
                        <button 
                          className="btn btn-outline-dark btn-sm d-flex align-items-center"
                          onClick={() => window.open('https://github.com/axfinn/todoIngPlus', '_blank')}
                        >
                          <i className="bi bi-star-fill me-1"></i> 
                          <span>{t('github.star')}</span>
                          {githubStats.stars > 0 && (
                            <span className="badge bg-secondary ms-1">{githubStats.stars}</span>
                          )}
                        </button>
                        <span className="d-flex align-items-center">
                          <i className="bi bi-git me-1"></i> 
                          {githubStats.forks > 0 && (
                            <span className="badge bg-secondary">{githubStats.forks}</span>
                          )}
                        </span>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>
          } />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/dashboard" element={
            <ProtectedRoute>
              <DashboardPage />
            </ProtectedRoute>
          } />
          <Route path="/events" element={<ProtectedRoute><EventsPage /></ProtectedRoute>} />
          <Route path="/events/:id" element={<ProtectedRoute><EventDetailPage /></ProtectedRoute>} />
          <Route path="/reminders" element={
            <ProtectedRoute>
              <RemindersPage />
            </ProtectedRoute>
          } />
          <Route path="/reports" element={
            <ProtectedRoute>
              <ReportsPage />
            </ProtectedRoute>
          } />
          <Route path="/unified" element={
            <ProtectedRoute>
              <UnifiedBoardPage />
            </ProtectedRoute>
          } />
        </Routes>
      </main>
  <footer className="bg-light py-3 mt-auto position-relative app-content-surface">
        <div className="container">
          <div className="text-center text-muted">
            <div className="d-flex align-items-center justify-content-center">
              <i className="bi bi-github me-2"></i>
              <a href="https://github.com/axfinn/todoIngPlus" target="_blank" rel="noopener noreferrer" className="text-decoration-none text-muted">
                {t('github.fork')}
              </a>
            </div>
            <div className="mt-2">
              &copy; {new Date().getFullYear()} {t('footer.copyright')}
            </div>
          </div>
        </div>
      </footer>
  </div>
  </NowProvider>
  );
};

export default App;