import type { ReactNode } from 'react';
import { clipExitDuration } from '../../hooks/useClipExits';

export default function ClipPresence({ children, exiting }: { children: ReactNode; exiting: boolean }) {
  return (
    <div ref={(node) => { if (node) node.inert = exiting; }}
      className={`grid grid-rows-[1fr] ${exiting ? 'animate-clip-exit pointer-events-none' : ''}`}
      style={exiting ? { animationDuration: `${clipExitDuration}ms` } : undefined}>
      <div className={exiting ? 'min-h-0 overflow-hidden' : 'min-h-0'}>{children}</div>
    </div>
  );
}
