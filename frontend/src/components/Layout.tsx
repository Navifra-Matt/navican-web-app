import React from 'react';
import Sidebar from './Sidebar';
import StatusBar from './StatusBar';

interface LayoutProps {
    children: React.ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
    return (
        <div className="flex h-screen bg-base-100 text-base-content font-sans overflow-hidden">
            <Sidebar />
            <div className="flex-1 flex flex-col min-w-0">
                <div className="flex-1 overflow-y-auto relative bg-base-200/50">
                    {children}
                </div>
                <StatusBar />
            </div>
        </div>
    );
};

export default Layout;


