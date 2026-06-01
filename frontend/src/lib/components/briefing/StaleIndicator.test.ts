import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import StaleIndicator from './StaleIndicator.svelte';

describe('StaleIndicator', () => {
  it('renders generating state with neutral styling', () => {
    render(StaleIndicator, { status: 'generating' });
    expect(document.querySelector('[role="status"]')).toBeInTheDocument();
    expect(document.querySelector('.stale-indicator')).toHaveClass('stale-indicator--neutral');
    expect(document.querySelector('.stale-indicator__label')).toHaveTextContent('Generating');
  });

  it('renders ready state with success styling', () => {
    render(StaleIndicator, { status: 'ready' });
    expect(document.querySelector('.stale-indicator')).toHaveClass('stale-indicator--success');
    expect(document.querySelector('.stale-indicator__label')).toHaveTextContent('Ready');
  });

  it('renders ready_with_caveats state with warning styling', () => {
    render(StaleIndicator, { status: 'ready_with_caveats' });
    expect(document.querySelector('.stale-indicator')).toHaveClass('stale-indicator--warning');
    expect(document.querySelector('.stale-indicator__label')).toHaveTextContent('Ready with caveats');
  });

  it('renders no_prior_memory state with neutral styling', () => {
    render(StaleIndicator, { status: 'no_prior_memory' });
    expect(document.querySelector('.stale-indicator')).toHaveClass('stale-indicator--neutral');
    expect(document.querySelector('.stale-indicator__label')).toHaveTextContent('No prior memory');
  });

  it('renders stale state with warning styling and status role', () => {
    render(StaleIndicator, { status: 'stale' });
    expect(document.querySelector('.stale-indicator')).toHaveClass('stale-indicator--warning');
    expect(document.querySelector('[role="status"]')).toBeInTheDocument();
    expect(document.querySelector('.stale-indicator__label')).toHaveTextContent('Stale');
  });

  it('renders failed state with danger styling and alert role', () => {
    render(StaleIndicator, { status: 'failed' });
    expect(document.querySelector('.stale-indicator')).toHaveClass('stale-indicator--danger');
    expect(document.querySelector('[role="alert"]')).toBeInTheDocument();
    expect(document.querySelector('.stale-indicator__label')).toHaveTextContent('Failed');
  });

  it('renders regenerating state with neutral styling', () => {
    render(StaleIndicator, { status: 'regenerating' });
    expect(document.querySelector('.stale-indicator')).toHaveClass('stale-indicator--neutral');
    expect(document.querySelector('.stale-indicator__label')).toHaveTextContent('Regenerating');
  });

  it('uses aria-live polite for status states', () => {
    render(StaleIndicator, { status: 'ready' });
    expect(document.querySelector('[aria-live="polite"]')).toBeInTheDocument();
  });

  it('uses aria-live assertive for failed state', () => {
    render(StaleIndicator, { status: 'failed' });
    expect(document.querySelector('[aria-live="assertive"]')).toBeInTheDocument();
  });

  it('renders aria-atomic true', () => {
    render(StaleIndicator, { status: 'generating' });
    expect(document.querySelector('[aria-atomic="true"]')).toBeInTheDocument();
  });
});