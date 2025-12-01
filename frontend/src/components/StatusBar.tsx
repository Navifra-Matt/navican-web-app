import React from 'react';
import useCanSocket from '../hooks/useCanSocket';

const StatusBar: React.FC = () => {
    const { isConnected } = useCanSocket('ws://localhost:8080');

    return (
        <div className="h-6 bg-primary text-primary-content flex items-center px-2 text-xs select-none justify-between">
            <div className="flex items-center gap-4">
                <div className="flex items-center gap-1">
                    <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-400' : 'bg-red-500'}`}></div>
                    <span>{isConnected ? 'Connected' : 'Disconnected'}</span>
                </div>
                <div className="flex items-center gap-1 opacity-80">
                    <span>ws://localhost:8080</span>
                </div>
            </div>

            <div className="flex items-center gap-4">
                <div className="flex items-center gap-1">
                    <span>Bus Load:</span>
                    <span className="font-mono">12%</span>
                </div>
                <div className="flex items-center gap-1">
                    <span>Errors:</span>
                    <span className="font-mono">0</span>
                </div>
                <div className="flex items-center gap-1">
                    <span>Mode:</span>
                    <span className="font-bold">OPERATIONAL</span>
                </div>
            </div>
        </div>
    );
};

export default StatusBar;
