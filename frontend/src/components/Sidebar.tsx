import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Monitor, ChartLine, FileText, FileOutput, Zap, Settings } from 'lucide-react';


const Sidebar: React.FC = () => {
    const location = useLocation();

    const isActive = (path: string) => location.pathname === path;

    const navItems = [
        {
            path: '/', label: 'Monitor', icon: <Monitor className="h-6 w-6" />
        },
        {
            path: '/motor', label: 'Motor Control', icon: <ChartLine className="h-6 w-6" />
        },
        {
            path: '/yaml-gen', label: 'Configuration', icon: <FileOutput className="h-6 w-6" />
        },
        {
            path: '/eds', label: 'EDS Viewer', icon: <FileText className="h-6 w-6" />
        },
    ];

    return (
        <div className="w-16 flex flex-col items-center py-4 bg-base-300 border-r border-base-100 z-20">
            <div className="mb-8 text-primary">
                <Zap className="h-8 w-8" />
            </div>

            <div className="flex flex-col gap-4 w-full">
                {navItems.map((item) => (
                    <Link
                        key={item.path}
                        to={item.path}
                        className={`flex flex-col items-center justify-center p-2 w-full transition-colors relative group ${isActive(item.path)
                            ? 'text-primary border-l-4 border-primary bg-base-100'
                            : 'text-base-content opacity-60 hover:opacity-100 hover:bg-base-200'
                            }`}
                        title={item.label}
                    >
                        {item.icon}
                    </Link>
                ))}
            </div>

            <div className="mt-auto flex flex-col gap-4 w-full">
                <button className="flex flex-col items-center justify-center p-2 w-full text-base-content opacity-60 hover:opacity-100 hover:bg-base-200 transition-colors">
                    <Settings className="h-6 w-6" />
                </button>
            </div>
        </div>
    );
};

export default Sidebar;
