import { useCallback, useEffect, useRef } from 'react';
import type { ReactNode } from 'react';
import { createPortal } from 'react-dom';

type CloseModal = (action?: () => void) => void;

interface ModalProps {
  label: string;
  backdropClassName: string;
  open?: boolean;
  onClose: () => void;
  children: (close: CloseModal) => ReactNode;
}

const exitTiming: KeyframeAnimationOptions = {
  duration: 180,
  easing: 'cubic-bezier(0.25, 0.7, 0.4, 1)',
  fill: 'forwards',
};

let mountedModals = 0;

export default function Modal({ label, backdropClassName, open = true, onClose, children }: ModalProps) {
  const backdropRef = useRef<HTMLDivElement>(null);
  const closing = useRef(false);

  const close = useCallback<CloseModal>((action) => {
    if (closing.current) return;
    closing.current = true;

    const backdrop = backdropRef.current;
    if (!backdrop || typeof backdrop.animate !== 'function' || window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      onClose();
    } else {
      // Start from the current styles so an immediate dismissal can interrupt the entrance smoothly.
      const panel = backdrop.querySelector('[data-modal-panel]');
      const animation = backdrop.animate([{ opacity: getComputedStyle(backdrop).opacity }, { opacity: 0 }], exitTiming);
      panel?.animate([
        { transform: getComputedStyle(panel).transform },
        { transform: 'translateY(6px) scale(0.985)' },
      ], exitTiming);
      const finish = () => {
        if (backdropRef.current === backdrop) onClose();
      };
      void animation.finished.then(finish, finish);
    }

    action?.();
  }, [onClose]);

  useEffect(() => {
    mountedModals += 1;
    document.body.classList.add('modal-open');
    return () => {
      mountedModals -= 1;
      if (mountedModals === 0) document.body.classList.remove('modal-open');
    };
  }, []);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      event.preventDefault();
      event.stopPropagation();
      close();
    };
    document.addEventListener('keydown', handleKeyDown, true);
    return () => document.removeEventListener('keydown', handleKeyDown, true);
  }, [close]);

  useEffect(() => {
    if (!open) close();
  }, [open, close]);

  return createPortal(
    <div
      ref={backdropRef}
      role="dialog"
      aria-modal="true"
      aria-label={label}
      className={`fixed inset-0 z-50 flex items-center justify-center p-4 motion-safe:animate-fade-in ${backdropClassName}`}
      onClick={(event) => {
        if (event.target === event.currentTarget) close();
      }}
    >
      {children(close)}
    </div>,
    document.body,
  );
}
