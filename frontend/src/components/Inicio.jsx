import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import './Inicio.css';

const Inicio = () => {
  const [loading, setLoading] = useState(false);

  const navigate = useNavigate();

  const handleEnterSystem = () => {
    setLoading(true);
    
    setTimeout(() => {
      navigate('/main');
      setLoading(false);
    }, 1500);
  };

  const renderLoadingOverlay = () => (
    loading && (
      <div className="loading-overlay">
        <div className="loading-content">
          <div className="loading-spinner"></div>
          <div className="loading-text">Cargando ExtreamFS...</div>
        </div>
      </div>
    )
  );

  const renderLogo = () => (
    <div className="logo-container">
      <div className="logo-frame">
        <div className="logo-icon">💾</div>
      </div>
    </div>
  );

  const renderTitle = () => (
    <div className="title-container">
      <h1 className="system-title">ExtreamFS</h1>
      <p className="system-subtitle">Sistema de Archivos Avanzado</p>
    </div>
  );

  const renderEnterButton = () => (
    <div className="button-container">
      <button 
        className="enter-button" 
        onClick={handleEnterSystem}
        disabled={loading}
      >
        <span className="button-text">Entrar al Sistema</span>
      </button>
    </div>
  );

  const renderFooter = () => (
    <div className="footer-container">
      <p className="developer-credit">
        Desarrollado por José Emanuel Monzón Lemus
      </p>
    </div>
  );

  return (
    <div className="inicio-container">
      {renderLoadingOverlay()}
      
      <div className="inicio-window">
        <div className="window-header">
          <div className="window-title">ExtreamFS - Inicio</div>
          <div className="window-controls">
            <div className="control-button minimize"></div>
            <div className="control-button maximize"></div>
            <div className="control-button close"></div>
          </div>
        </div>
        
        <div className="window-content">
          {renderLogo()}
          {renderTitle()}
          {renderEnterButton()}
          {renderFooter()}
        </div>
        
        <div className="window-statusbar">
          <div className="status-text">Listo</div>
          <div className="status-time" id="current-time">
            {new Date().toLocaleTimeString()}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Inicio;