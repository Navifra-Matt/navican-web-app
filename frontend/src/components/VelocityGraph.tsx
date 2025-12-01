import React, { useEffect, useState, useRef, useMemo } from 'react';
import uPlot from 'uplot';
import UplotReact from 'uplot-react';
import 'uplot/dist/uPlot.min.css';

interface VelocityGraphProps {
    targetVelocity: number;
    actualVelocity: number;
}

const VelocityGraph: React.FC<VelocityGraphProps> = ({ targetVelocity, actualVelocity }) => {
    const containerRef = useRef<HTMLDivElement>(null);
    const [dimensions, setDimensions] = useState({ width: 0, height: 0 });
    const maxPoints = 200;
    const updateInterval = 20; // ms

    // Data state: [Time[], Target[], Actual[]]
    const [data, setData] = useState<[number[], number[], number[]]>([[], [], []]);

    // Refs for latest values to avoid closure staleness in interval
    const latestValues = useRef({ target: targetVelocity, actual: actualVelocity });

    useEffect(() => {
        latestValues.current = { target: targetVelocity, actual: actualVelocity };
    }, [targetVelocity, actualVelocity]);

    // Resize Observer
    useEffect(() => {
        if (!containerRef.current) return;

        const resizeObserver = new ResizeObserver(entries => {
            if (entries[0]) {
                const { width, height } = entries[0].contentRect;
                setDimensions({ width, height });
            }
        });

        resizeObserver.observe(containerRef.current);
        return () => resizeObserver.disconnect();
    }, []);

    // Data Update Loop
    useEffect(() => {
        // Initialize data
        const initialTime = Array.from({ length: maxPoints }, (_, i) => i);
        const initialTarget = Array(maxPoints).fill(0);
        const initialActual = Array(maxPoints).fill(0);

        setData([initialTime, initialTarget, initialActual]);

        let tick = maxPoints;

        const interval = setInterval(() => {
            setData(prevData => {
                const [time, target, actual] = prevData;

                const newTime = [...time.slice(1), tick++];
                const newTarget = [...target.slice(1), latestValues.current.target];
                const newActual = [...actual.slice(1), latestValues.current.actual];

                return [newTime, newTarget, newActual];
            });
        }, updateInterval);

        return () => clearInterval(interval);
    }, []);

    const options = useMemo<uPlot.Options>(() => ({
        title: "Velocity Tracking",
        width: dimensions.width,
        height: dimensions.height - 2, // Subtract buffer to prevent overflow
        series: [
            {}, // x-axis (Time)
            {
                label: "Target",
                stroke: "#4ade80",
                width: 2,
                points: { show: false }
            },
            {
                label: "Actual",
                stroke: "#f87171",
                width: 2,
                points: { show: false }
            }
        ],
        scales: {
            x: { time: false },
            y: {
                auto: false,
                range: [-1200, 1200],
            }
        },
        axes: [
            { show: false }, // Hide X axis
            {
                show: true,
                stroke: "#888",
                grid: { show: true, stroke: "#333", width: 1, dash: [5, 5] },
                ticks: { show: true, stroke: "#888", dash: [5, 5] }
            }
        ],
        legend: { show: false } // We have our own legend or can use uPlot's
    }), [dimensions]);

    return (
        <div ref={containerRef} className="absolute inset-0 overflow-hidden">
            {dimensions.width > 0 && dimensions.height > 0 && (
                <UplotReact
                    options={options}
                    data={data}
                />
            )}

            {/* Custom Legend Overlay */}
            <div className="absolute top-2 right-2 flex gap-4 text-xs font-mono bg-base-100/80 p-1 rounded backdrop-blur-sm pointer-events-none">
                <div className="flex items-center gap-1">
                    <div className="w-3 h-1 rounded-full" style={{ backgroundColor: '#4ade80' }}></div>
                    <span>Target</span>
                </div>
                <div className="flex items-center gap-1">
                    <div className="w-3 h-1 rounded-full" style={{ backgroundColor: '#f87171' }}></div>
                    <span>Actual</span>
                </div>
            </div>
        </div>
    );
};

export default VelocityGraph;
