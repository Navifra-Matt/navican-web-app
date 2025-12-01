import React from 'react';
import { Monitor } from 'lucide-react';
import useCanSocket from '../hooks/useCanSocket';
import ReceiveWindow from './ReceiveWindow';
import TransmitWindow from './TransmitWindow';

const Dashboard: React.FC = () => {
    // Using localhost:8080 as default, this could be an env var
    const { messages, sendMessage } = useCanSocket('ws://localhost:8080');

    return (
        <div className="flex flex-col h-full bg-base-200">
            {/* Header */}
            <div className="bg-base-100 border-b border-base-300 p-4 flex justify-between items-center shadow-sm z-10">
                <h1 className="text-xl font-bold text-primary flex items-center gap-2">
                    <Monitor className="h-6 w-6" />
                    CAN Monitor
                </h1>
            </div>

            {/* Main Content */}
            <div className="flex-1 flex flex-col gap-4 min-h-0 p-6">
                {/* Top: Receive Window (60% height) */}
                <div className="flex-[3] min-h-0">
                    <ReceiveWindow messages={messages} />
                </div>

                {/* Bottom: Transmit Window (40% height) */}
                <div className="flex-[2] min-h-0">
                    <TransmitWindow sendMessage={sendMessage} />
                </div>
            </div>
        </div>
    );
};

export default Dashboard;
