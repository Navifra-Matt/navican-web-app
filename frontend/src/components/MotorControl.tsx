import React, { useState } from 'react';
import { ChartLine } from 'lucide-react';
import VelocityGraph from './VelocityGraph';

const MotorControl: React.FC = () => {
    const [targetVelocity, setTargetVelocity] = useState(0);
    const [actualVelocity, setActualVelocity] = useState(0);
    const [isEnabled, setIsEnabled] = useState(false);

    // Simulate motor inertia
    React.useEffect(() => {
        const interval = setInterval(() => {
            setActualVelocity(prev => {
                const diff = targetVelocity - prev;
                if (Math.abs(diff) < 1) return targetVelocity;
                return prev + diff * 0.1; // Simple easing factor
            });
        }, 20);
        return () => clearInterval(interval);
    }, [targetVelocity]);

    return (
        <div className="flex flex-col h-full bg-base-200">
            {/* Header */}
            <div className="bg-base-100 border-b border-base-300 p-4 flex justify-between items-center shadow-sm z-10">
                <h1 className="text-xl font-bold text-primary flex items-center gap-2">
                    <ChartLine className="h-6 w-6" />
                    Motor Control
                </h1>
                <div className="flex gap-2">
                    <button className={`btn btn-sm ${isEnabled ? 'btn-error' : 'btn-success'}`} onClick={() => setIsEnabled(!isEnabled)}>
                        {isEnabled ? 'DISABLE' : 'ENABLE'}
                    </button>
                    <button className="btn btn-sm btn-outline">Reset Faults</button>
                </div>
            </div>

            {/* Main Content */}
            <div className="flex-1 overflow-auto p-6 flex flex-col gap-6">

                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    {/* Status Panel */}
                    <div className="card bg-base-200 shadow-xl">
                        <div className="card-body p-4">
                            <h3 className="card-title text-sm opacity-70">Status</h3>
                            <div className="grid grid-cols-2 gap-4 mt-2">
                                <div className="stat p-2 bg-base-100 rounded-box">
                                    <div className="stat-title text-xs">Voltage</div>
                                    <div className="stat-value text-lg text-warning">48.2 V</div>
                                </div>
                                <div className="stat p-2 bg-base-100 rounded-box">
                                    <div className="stat-title text-xs">Current</div>
                                    <div className="stat-value text-lg text-info">2.4 A</div>
                                </div>
                                <div className="stat p-2 bg-base-100 rounded-box">
                                    <div className="stat-title text-xs">Temperature</div>
                                    <div className="stat-value text-lg text-error">42°C</div>
                                </div>
                                <div className="stat p-2 bg-base-100 rounded-box">
                                    <div className="stat-title text-xs">Position</div>
                                    <div className="stat-value text-lg">12405</div>
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Control Panel */}
                    <div className="card bg-base-200 shadow-xl md:col-span-2">
                        <div className="card-body p-4">
                            <h3 className="card-title text-sm opacity-70">Velocity Control</h3>
                            <div className="flex flex-col gap-4 mt-4">
                                <input
                                    type="range"
                                    min="-1000"
                                    max="1000"
                                    value={targetVelocity}
                                    onChange={(e) => setTargetVelocity(parseInt(e.target.value))}
                                    className="range range-primary range-lg w-full"
                                    step="10"
                                />
                                <div className="flex justify-between text-xs font-mono opacity-50 w-full mt-1">
                                    <span>-1000 RPM</span>
                                    <span>0 RPM</span>
                                    <span>+1000 RPM</span>
                                </div>
                                <div className="flex justify-center">
                                    <div className="stats shadow bg-base-100">
                                        <div className="stat place-items-center">
                                            <div className="stat-title">Target Velocity</div>
                                            <div className="stat-value text-primary">{targetVelocity} <span className="text-sm">RPM</span></div>
                                        </div>
                                        <div className="stat place-items-center">
                                            <div className="stat-title">Actual Velocity</div>
                                            <div className="stat-value">{Math.round(actualVelocity)} <span className="text-sm">RPM</span></div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Graphs Placeholder */}
                <div className="card bg-base-200 shadow-xl h-[400px] shrink-0">
                    <div className="card-body p-4">
                        <h3 className="card-title text-sm opacity-70">Real-time Graphs</h3>
                        <div className="flex-1 bg-base-100 rounded-box overflow-hidden border border-base-300 relative min-h-0">
                            <VelocityGraph targetVelocity={targetVelocity} actualVelocity={actualVelocity} />
                        </div>
                    </div>
                </div>
            </div>

        </div>
    );
};

export default MotorControl;
