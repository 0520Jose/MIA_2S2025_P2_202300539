import React, { useState } from 'react';
import Cookies from 'js-cookie';
import { useNavigate} from 'react-router-dom';
import './Login.css';

const Login = () => {
  const [partitionId, setPartitionId] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [remember, setRemember] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsLoading(true);
    alert(username);
    Cookies.set('usuario', username, { expires: 7 });
    navigate('/main');
    setTimeout(() => {
      alert(`ID: ${partitionId}\nUsuario: ${username}\nContraseña: ${password}\nRecordar: ${remember}`);
      setIsLoading(false);
    }, 1000);
  };

  return (
    <div className="login-main-container">
      <div className="login-header">
        <div className="login-logo-container">
          <div className="login-logo">
            <div className="login-logo-inner"></div>
          </div>
        </div>
        <h2 className="login-title">
          Iniciar sesión
        </h2>
        <p className="login-subtitle">
          Accede a tu cuenta para continuar
        </p>
      </div>

      <div className="login-form-container">
        <div className="login-card">
          <form className="login-form" onSubmit={handleSubmit}>
            <div className="form-group">
              <label htmlFor="partitionId" className="form-label">
                ID Partición
              </label>
              <input
                id="partitionId"
                name="partitionId"
                type="text"
                autoComplete="username"
                required
                value={partitionId}
                onChange={(e) => setPartitionId(e.target.value)}
                className="form-input"
                placeholder="Ej: 391A"
              />
            </div>
            <div className="form-group">
              <label htmlFor="username" className="form-label">
                Usuario
              </label>
              <input
                id="username"
                name="username"
                type="text"
                autoComplete="username"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="form-input"
                placeholder="Ingresa tu usuario"
              />
            </div>
            <div className="form-group">
              <label htmlFor="password" className="form-label">
                Contraseña
              </label>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="form-input"
                placeholder="••••••••"
              />
            </div>
            <div className="remember-container">
              <div className="checkbox-group">
                <input
                  id="remember"
                  name="remember"
                  type="checkbox"
                  checked={remember}
                  onChange={() => setRemember(!remember)}
                  className="form-checkbox"
                />
                <label htmlFor="remember" className="checkbox-label">
                  Recordar usuario
                </label>
              </div>
            </div>
            <div>
              <button
                type="submit"
                disabled={isLoading}
                className="submit-button"
              >
                {isLoading ? (
                  <div className="loading-container">
                    <svg className="loading-spinner" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    Iniciando sesión...
                  </div>
                ) : (
                  'Iniciar sesión'
                )}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default Login;