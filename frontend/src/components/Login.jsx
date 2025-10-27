import React, { useState } from 'react';
import Cookies from 'js-cookie';
import { useNavigate } from 'react-router-dom';
import './Login.css';

const Login = () => {
  const [partitionId, setPartitionId] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [remember, setRemember] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    if (!partitionId || !username || !password) {
      setError('Todos los campos son obligatorios');
      setIsLoading(false);
      return;
    }

    try {
      const loginCommand = `login -user=${username} -pass=${password} -id=${partitionId}`;
      
      const response = await fetch('http://ec2-18-223-185-41.us-east-2.compute.amazonaws.com:8000/execute', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ 
          comando: loginCommand 
        }),
      });

      const data = await response.json();
      
      if (data.salida && data.salida.includes('Error')) {
        setError('Credenciales incorrectas o partición no existe');
        setIsLoading(false);
        return;
      }

      Cookies.set('usuario', username, { expires: remember ? 7 : 1 });
      Cookies.set('partitionId', partitionId, { expires: remember ? 7 : 1 });
      
      setTimeout(() => {
        navigate('/main');
        setIsLoading(false);
      }, 1000);

    } catch (error) {
      console.error('Error en login:', error);
      setError('Error de conexión con el servidor');
      setIsLoading(false);
    }
  };

  const handleCancel = () => {
    navigate('/main');
  };

  return (
    <div className="login-bg">
      <div className="login-window">
        <div className="login-header">
          <div className="login-title-bar">
            <div className="login-title-icon">🔒</div>
            <div className="login-title-text">Iniciar Sesión - ExtreamFS</div>
            <div className="login-title-controls">
              <button className="login-close-btn" onClick={handleCancel}>×</button>
            </div>
          </div>
        </div>

        <div className="login-content">
          <div className="login-logo-section">
            <div className="login-logo">
              <div className="login-logo-disk">💾</div>
            </div>
            <div className="login-system-info">
              <h3>ExtreamFS</h3>
              <p>Sistema de Archivos Avanzado</p>
            </div>
          </div>

          <form className="login-form" onSubmit={handleSubmit}>
            {error && (
              <div className="login-error">
                <div className="error-icon">⚠</div>
                <div className="error-text">{error}</div>
              </div>
            )}

            <div className="form-group">
              <label htmlFor="partitionId" className="form-label">
                ID de Partición:
              </label>
              <input
                id="partitionId"
                name="partitionId"
                type="text"
                required
                value={partitionId}
                onChange={(e) => setPartitionId(e.target.value.toUpperCase())}
                className="form-input"
                placeholder="Ej: 391A"
                disabled={isLoading}
              />
            </div>

            <div className="form-group">
              <label htmlFor="username" className="form-label">
                Usuario:
              </label>
              <input
                id="username"
                name="username"
                type="text"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="form-input"
                placeholder="Ingresa tu usuario"
                disabled={isLoading}
              />
            </div>

            <div className="form-group">
              <label htmlFor="password" className="form-label">
                Contraseña:
              </label>
              <input
                id="password"
                name="password"
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="form-input"
                placeholder="••••••••"
                disabled={isLoading}
              />
            </div>

            <div className="form-options">
              <label className="checkbox-container">
                <input
                  type="checkbox"
                  checked={remember}
                  onChange={() => setRemember(!remember)}
                  disabled={isLoading}
                />
                <span className="checkmark"></span>
                Recordar usuario
              </label>
            </div>

            <div className="form-buttons">
              <button
                type="button"
                onClick={handleCancel}
                className="btn-cancel"
                disabled={isLoading}
              >
                Cancelar
              </button>
              <button
                type="submit"
                disabled={isLoading}
                className="btn-login"
              >
                {isLoading ? (
                  <div className="login-loading">
                    <div className="login-spinner"></div>
                    Iniciando sesión...
                  </div>
                ) : (
                  'Iniciar Sesión'
                )}
              </button>
            </div>
          </form>

          <div className="login-footer">
            <div className="login-help">
              <button type="button" className="btn-help">
                ?
              </button>
              <span>Ayuda</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Login;