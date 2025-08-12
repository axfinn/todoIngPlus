import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { Provider } from 'react-redux';
import { store } from './app/store';
import './app/i18n';

// 引入全局样式（包含 body::before 背景 / 卡片 / 动画 等）
import './index.css';

// 引入Bootstrap JavaScript功能
import 'bootstrap/dist/js/bootstrap.bundle.min.js';

// 引入本地Bootstrap和Bootstrap Icons CSS
import 'bootstrap/dist/css/bootstrap.min.css';
import 'bootstrap-icons/font/bootstrap-icons.css';
import './styles/focus.css';

import App from './App';
import ToastProvider from './components/ToastProvider';
import { GlobalErrorBoundary } from './components/GlobalErrorBoundary';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <Provider store={store}>
      <BrowserRouter>
        <ToastProvider>
          <GlobalErrorBoundary>
            <App />
          </GlobalErrorBoundary>
        </ToastProvider>
      </BrowserRouter>
    </Provider>
  </React.StrictMode>,
);