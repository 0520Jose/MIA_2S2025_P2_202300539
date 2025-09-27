import React from 'react';
import Cookies from 'js-cookie';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import Login from '../components/Login';
import Main from '../components/Main';
import FileExplorer from '../components/Explorer';
import App from '../App';

const AppRouter = () => {
  const isAuthenticated = !!Cookies.get('usuario'); 

  return (
    <Router>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/main" element={<Main />} />
        <Route path="/explorer" element={isAuthenticated ? <FileExplorer /> : <Navigate to="/login" />} />
        <Route path="/" element={isAuthenticated ? <App /> : <Navigate to="/main" />} />
      </Routes>
    </Router>
  );
};

export default AppRouter;
