import React, { useState, useEffect } from 'react';
import { HardDrive, Folder, File, ArrowLeft, Home, Monitor, Database } from 'lucide-react';
import './Explorer.css';

const FileExplorer = () => {
  const [disks, setDisks] = useState([]);
  const [selectedDisk, setSelectedDisk] = useState(null);
  const [partitions, setPartitions] = useState([]);
  const [selectedPartition, setSelectedPartition] = useState(null);
  const [currentPath, setCurrentPath] = useState('/');
  const [files, setFiles] = useState([]);
  const [selectedFile, setSelectedFile] = useState(null);
  const [fileContent, setFileContent] = useState('');

  useEffect(() => {
    fetch('/api/disks')
      .then(res => res.json())
      .then(setDisks)
      .catch(() => setDisks([]));
  }, []);

  useEffect(() => {
    if (selectedDisk) {
      fetch(`/api/partitions?path=${encodeURIComponent(selectedDisk.path)}`)
        .then(res => res.json())
        .then(setPartitions)
        .catch(() => setPartitions([]));
    }
  }, [selectedDisk]);

  useEffect(() => {
    if (selectedDisk && selectedPartition) {
      fetch(`/api/files?disk=${encodeURIComponent(selectedDisk.path)}&partition=${encodeURIComponent(selectedPartition.name)}&path=${encodeURIComponent(currentPath)}`)
        .then(res => res.json())
        .then(setFiles)
        .catch(() => setFiles([]));
    }
  }, [selectedDisk, selectedPartition, currentPath]);

  useEffect(() => {
    if (selectedFile && selectedDisk && selectedPartition) {
      fetch(`/api/files?disk=${encodeURIComponent(selectedDisk.path)}&partition=${encodeURIComponent(selectedPartition.name)}&path=${encodeURIComponent(currentPath + (currentPath.endsWith('/') ? '' : '/') + selectedFile)}`)
        .then(res => res.json())
        .then(data => setFileContent(data.content || ''))
        .catch(() => setFileContent('Error al cargar archivo'));
    }
  }, [selectedFile]);

 if (!selectedDisk) {
  return (
    <div className="ext-explorer-container">
      <div className="ext-container">
        <div className="ext-header">
          <div className="ext-icon-wrapper">
            <HardDrive className="ext-main-icon" />
          </div>
          <h1 className="ext-title">Visualizador del sistema de archivos</h1>
          <p className="ext-subtitle">Explorador de sistemas de archivos Linux EXT2 y EXT3</p>
        </div>
        <div className="ext-disk-grid">
          {disks.map(disk => (
            <div key={disk.path} className="ext-disk-card">
              <div className="ext-card-content">
                <div className="ext-card-header">
                  <div className="ext-disk-info">
                    <div className="ext-disk-icon ext-hdd">
                      <HardDrive className="ext-icon" />
                    </div>
                    <div>
                      <h3 className="ext-disk-name">{disk.name}</h3>
                      <p className="ext-disk-details">{disk.path}</p>
                      {/* Información básica del disco */}
                      <p className="ext-disk-meta">
                        Capacidad: {disk.size} <br />
                        Fit: {disk.fit} <br />
                        Particiones montadas: {disk.mountedPartitions?.length || 0}
                      </p>
                    </div>
                  </div>
                  <span className="ext-linux-badge">Linux</span>
                </div>
                <div className="ext-card-footer">
                  <button
                    onClick={() => setSelectedDisk(disk)}
                    className="ext-button ext-button-primary"
                  >
                    Explorar
                  </button>
                </div>
              </div>
            </div>
          ))}
          {disks.length === 0 && <div>No hay discos encontrados.</div>}
        </div>
      </div>
    </div>
  );
}

if (!selectedPartition) {
  return (
    <div className="ext-explorer-container">
      <div className="ext-container">
        <div className="ext-nav-header">
          <button onClick={() => setSelectedDisk(null)} className="ext-back-button">
            <ArrowLeft className="ext-back-icon" />
          </button>
          <div>
            <h1 className="ext-page-title">Particiones de {selectedDisk.name}</h1>
            <p className="ext-page-subtitle">Selecciona una partición EXT2/EXT3 para explorar su contenido</p>
          </div>
        </div>
        <div className="ext-partition-grid">
          {partitions.map(partition => (
            <div key={partition.name} className="ext-partition-card">
              <div className="ext-card-content">
                <div className="ext-partition-header">
                  <div className="ext-partition-info">
                    <div className={`ext-partition-icon`}>
                      <Monitor className="ext-icon" />
                    </div>
                    <div>
                      <h3 className="ext-partition-name">{partition.name}</h3>
                      <p className="ext-partition-details">{partition.type}</p>
                      {/* Información básica de la partición */}
                      <p className="ext-partition-meta">
                        Tamaño: {partition.size} <br />
                        Fit: {partition.fit} <br />
                        Estado: {partition.status}
                      </p>
                    </div>
                  </div>
                  <span className="ext-status-badge ext-status-mounted">{partition.status}</span>
                </div>
                <button
                  onClick={() => {
                    setSelectedPartition(partition);
                    setCurrentPath('/');
                  }}
                  className="ext-button ext-button-primary ext-button-full"
                >
                  Explorar Archivos
                </button>
              </div>
            </div>
          ))}
          {partitions.length === 0 && <div>No hay particiones encontradas.</div>}
        </div>
      </div>
    </div>
  );
}

  if (selectedFile) {
    return (
      <div className="ext-explorer-container">
        <div className="ext-container">
          <div className="ext-nav-header">
            <button onClick={() => setSelectedFile(null)} className="ext-back-button">
              <ArrowLeft className="ext-back-icon" />
            </button>
            <div className="ext-file-header">
              <div className="ext-file-icon-wrapper">
                <File className="ext-file-icon" />
              </div>
              <div>
                <h1 className="ext-page-title">{selectedFile}</h1>
              </div>
            </div>
          </div>
          <div className="ext-file-container">
            <div className="ext-file-card">
              <div className="ext-file-card-header">
                <h3 className="ext-file-card-title">Contenido del archivo</h3>
              </div>
              <div className="ext-file-card-content">
                <pre className="ext-file-content">{fileContent}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="ext-explorer-container">
      <div className="ext-container">
        <div className="ext-explorer-header">
          <div className="ext-nav-header">
            <button onClick={() => setSelectedPartition(null)} className="ext-back-button">
              <ArrowLeft className="ext-back-icon" />
            </button>
            <div>
              <h1 className="ext-page-title">Explorador de Archivos</h1>
              <div className="ext-breadcrumb">
                {currentPath.split('/').filter(Boolean).map((part, idx, arr) => (
                  <span key={idx} className={idx === arr.length - 1 ? 'ext-breadcrumb-current' : 'ext-breadcrumb-item'}>
                    {part}
                    {idx < arr.length - 1 && <span className="ext-breadcrumb-separator">/</span>}
                  </span>
                ))}
              </div>
            </div>
          </div>
          {currentPath !== '/' && (
            <button onClick={() => {
              const parts = currentPath.split('/').filter(Boolean);
              parts.pop();
              setCurrentPath(parts.length ? '/' + parts.join('/') : '/');
            }} className="ext-button ext-button-secondary">
              Subir
            </button>
          )}
        </div>
        <div className="ext-file-explorer">
          <div className="ext-file-explorer-card">
            <div className="ext-file-explorer-header">
              <div className="ext-partition-indicator">
                <Home className="ext-home-icon" />
                <span className="ext-partition-name">{selectedPartition.name}</span>
              </div>
            </div>
            <div className="ext-file-explorer-content">
              {files.length === 0 ? (
                <div className="ext-empty-folder">
                  <div className="ext-empty-folder-icon">
                    <Folder className="ext-empty-icon" />
                  </div>
                  <h3 className="ext-empty-folder-title">Directorio vacío</h3>
                  <p className="ext-empty-folder-subtitle">
                    No hay archivos o directorios en esta ubicación
                  </p>
                </div>
              ) : (
                <div className="ext-file-grid">
                  {files.map(node => (
                    <div
                      key={node.name}
                      onClick={() => {
                        if (node.type === 'folder') {
                          setCurrentPath(currentPath === '/' ? `/${node.name}` : `${currentPath}/${node.name}`);
                        } else {
                          setSelectedFile(node.name);
                        }
                      }}
                      className="ext-file-item"
                    >
                      <div className="ext-file-item-content">
                        <div className={`ext-file-item-icon ${node.type === 'folder' ? 'ext-folder-icon' : 'ext-file-icon-item'}`}>
                          {node.type === 'folder' ? <Folder className="ext-icon" /> : <File className="ext-icon" />}
                        </div>
                        <div className="ext-file-item-details">
                          <h4 className="ext-file-item-name">{node.name}</h4>
                          <p className="ext-file-item-meta">
                            {node.type === 'folder' ? 'Directorio' : `${node.size} • ${node.permissions}`}
                          </p>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default FileExplorer;