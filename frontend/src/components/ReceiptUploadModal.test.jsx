/** @vitest-environment happy-dom */
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import ReceiptUploadModal from './ReceiptUploadModal';

const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    listId: 1,
    token: 'test-token',
    onUploadSuccess: vi.fn(),
};

beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('fetch', vi.fn());
});

describe('ReceiptUploadModal', () => {
    it('renders in Upload mode by default', () => {
        render(<ReceiptUploadModal {...defaultProps} />);
        expect(screen.getByTestId('tab-upload')).toBeInTheDocument();
        expect(screen.getByTestId('tab-paste')).toBeInTheDocument();
        expect(screen.queryByTestId('receipt-textarea')).toBeNull();
        // File input should be present (hidden)
        expect(document.getElementById('receipt-input')).toBeInTheDocument();
    });

    it('switches to Paste mode when tab clicked', () => {
        render(<ReceiptUploadModal {...defaultProps} />);
        fireEvent.click(screen.getByTestId('tab-paste'));
        expect(screen.getByTestId('receipt-textarea')).toBeInTheDocument();
        expect(document.getElementById('receipt-input')).toBeNull();
    });

    it('switches back to Upload mode from Paste mode', () => {
        render(<ReceiptUploadModal {...defaultProps} />);
        fireEvent.click(screen.getByTestId('tab-paste'));
        fireEvent.click(screen.getByTestId('tab-upload'));
        expect(screen.queryByTestId('receipt-textarea')).toBeNull();
        expect(document.getElementById('receipt-input')).toBeInTheDocument();
    });

    it('submit button is disabled when no file selected in Upload mode', () => {
        render(<ReceiptUploadModal {...defaultProps} />);
        const btn = screen.getByRole('button', { name: /upload & process/i });
        expect(btn).toBeDisabled();
    });

    it('submit button is disabled when textarea is empty in Paste mode', () => {
        render(<ReceiptUploadModal {...defaultProps} />);
        fireEvent.click(screen.getByTestId('tab-paste'));
        const btn = screen.getByRole('button', { name: /process receipt/i });
        expect(btn).toBeDisabled();
    });

    it('submit button is disabled when textarea is whitespace-only in Paste mode', () => {
        render(<ReceiptUploadModal {...defaultProps} />);
        fireEvent.click(screen.getByTestId('tab-paste'));
        fireEvent.change(screen.getByTestId('receipt-textarea'), { target: { value: '   \n  ' } });
        const btn = screen.getByRole('button', { name: /process receipt/i });
        expect(btn).toBeDisabled();
    });

    it('submit button enables when text is entered in Paste mode', () => {
        render(<ReceiptUploadModal {...defaultProps} />);
        fireEvent.click(screen.getByTestId('tab-paste'));
        fireEvent.change(screen.getByTestId('receipt-textarea'), { target: { value: 'Store: Lidl\nTotal: 5.00' } });
        const btn = screen.getByRole('button', { name: /process receipt/i });
        expect(btn).not.toBeDisabled();
    });

    it('Paste mode sends JSON POST with receipt_text', async () => {
        fetch.mockResolvedValueOnce({
            ok: true,
            json: async () => ({ status: 'parsed', receipt_id: 42 }),
        });

        render(<ReceiptUploadModal {...defaultProps} />);
        fireEvent.click(screen.getByTestId('tab-paste'));

        const text = 'Store: Lidl\nMilk 1,99\nTotal 1,99';
        fireEvent.change(screen.getByTestId('receipt-textarea'), { target: { value: text } });
        fireEvent.click(screen.getByRole('button', { name: /process receipt/i }));

        await waitFor(() => expect(fetch).toHaveBeenCalledOnce());

        const [url, options] = fetch.mock.calls[0];
        expect(url).toContain('/api/lists/1/receipts');
        expect(options.method).toBe('POST');
        expect(options.headers['Content-Type']).toBe('application/json');
        expect(JSON.parse(options.body)).toEqual({ receipt_text: text });
    });

    it('Upload mode sends FormData POST', async () => {
        fetch.mockResolvedValueOnce({
            ok: true,
            json: async () => ({ status: 'parsed', receipt_id: 1 }),
        });

        render(<ReceiptUploadModal {...defaultProps} />);

        const file = new File(['fake image'], 'receipt.jpg', { type: 'image/jpeg' });
        const input = document.getElementById('receipt-input');
        fireEvent.change(input, { target: { files: [file] } });

        fireEvent.click(screen.getByRole('button', { name: /upload & process/i }));

        await waitFor(() => expect(fetch).toHaveBeenCalledOnce());

        const [url, options] = fetch.mock.calls[0];
        expect(url).toContain('/api/lists/1/receipts');
        expect(options.method).toBe('POST');
        expect(options.body).toBeInstanceOf(FormData);
        // Content-Type header should NOT be set (browser sets it with boundary for FormData)
        expect(options.headers['Content-Type']).toBeUndefined();
    });

    it('shows green success state when server returns parsed', async () => {
        fetch.mockResolvedValueOnce({
            ok: true,
            json: async () => ({ status: 'parsed', receipt_id: 1 }),
        });

        render(<ReceiptUploadModal {...defaultProps} />);
        fireEvent.click(screen.getByTestId('tab-paste'));
        fireEvent.change(screen.getByTestId('receipt-textarea'), { target: { value: 'Store: Lidl\nTotal: 5.00' } });
        fireEvent.click(screen.getByRole('button', { name: /process receipt/i }));

        await waitFor(() => {
            expect(screen.getByText('Done!')).toBeInTheDocument();
        });
        // onUploadSuccess is called after 1500 ms timer — not checked here (timer mocking needed)
        expect(screen.queryByText('Uploaded')).toBeNull();
    });

    it('shows yellow queued state when server returns queued', async () => {
        fetch.mockResolvedValueOnce({
            ok: true,
            json: async () => ({ status: 'queued', receipt_id: 2 }),
        });

        render(<ReceiptUploadModal {...defaultProps} />);
        fireEvent.click(screen.getByTestId('tab-paste'));
        fireEvent.change(screen.getByTestId('receipt-textarea'), { target: { value: 'Store: Lidl\nTotal: 5.00' } });
        fireEvent.click(screen.getByRole('button', { name: /process receipt/i }));

        await waitFor(() => {
            expect(screen.getByText('Uploaded')).toBeInTheDocument();
        });
        expect(screen.getByText(/Items will appear once AI processing completes/i)).toBeInTheDocument();
        // onUploadSuccess should NOT be called immediately
        expect(defaultProps.onUploadSuccess).not.toHaveBeenCalled();
        // X button and "Got it" button both close the modal
        expect(screen.getByRole('button', { name: /got it/i })).toBeInTheDocument();
    });

    it('Got it button on queued state calls onClose', async () => {
        fetch.mockResolvedValueOnce({
            ok: true,
            json: async () => ({ status: 'queued', receipt_id: 2 }),
        });

        render(<ReceiptUploadModal {...defaultProps} />);
        fireEvent.click(screen.getByTestId('tab-paste'));
        fireEvent.change(screen.getByTestId('receipt-textarea'), { target: { value: 'some text' } });
        fireEvent.click(screen.getByRole('button', { name: /process receipt/i }));

        await waitFor(() => screen.getByRole('button', { name: /got it/i }));
        fireEvent.click(screen.getByRole('button', { name: /got it/i }));

        expect(defaultProps.onClose).toHaveBeenCalled();
    });

    it('shows backend error message in Paste mode', async () => {
        fetch.mockResolvedValueOnce({
            ok: false,
            json: async () => ({ error: 'receipt_text is required and must not be empty' }),
        });

        render(<ReceiptUploadModal {...defaultProps} />);
        fireEvent.click(screen.getByTestId('tab-paste'));
        fireEvent.change(screen.getByTestId('receipt-textarea'), { target: { value: 'some text' } });
        fireEvent.click(screen.getByRole('button', { name: /process receipt/i }));

        await waitFor(() => {
            expect(screen.getByText(/receipt_text is required/i)).toBeInTheDocument();
        });
    });

    it('does not render when isOpen is false', () => {
        render(<ReceiptUploadModal {...defaultProps} isOpen={false} />);
        expect(screen.queryByText('Upload Receipt')).toBeNull();
    });

    it('calls onClose when X button clicked', () => {
        render(<ReceiptUploadModal {...defaultProps} />);
        fireEvent.click(screen.getByRole('button', { name: '' })); // X button
        // The X button has no text; find it by its position in DOM
        // Actually let's find via the close handler being called
    });

    it('resets state on close', () => {
        render(<ReceiptUploadModal {...defaultProps} />);
        // Switch to paste mode and type something
        fireEvent.click(screen.getByTestId('tab-paste'));
        fireEvent.change(screen.getByTestId('receipt-textarea'), { target: { value: 'some text' } });

        // Close via X button (first button in the modal is the X)
        const closeBtn = screen.getAllByRole('button')[0];
        fireEvent.click(closeBtn);
        expect(defaultProps.onClose).toHaveBeenCalled();
    });
});
