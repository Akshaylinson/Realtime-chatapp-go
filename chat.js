document.addEventListener('DOMContentLoaded', function() {
    const chatWindow = document.getElementById('chat-window');
    const messageForm = document.getElementById('message-form');
    const messageInput = document.getElementById('message-input');
    const sendButton = document.getElementById('send-button');
    const usernameInput = document.getElementById('username-input');
    const setUsernameBtn = document.getElementById('set-username');
    const totalMessagesSpan = document.getElementById('total-messages');
    const activeUsersSpan = document.getElementById('active-users');
    const connectionStatusSpan = document.getElementById('connection-status');
    
    let ws;
    let username = '';
    let isConnected = false;

    // Update stats periodically
    setInterval(updateStats, 5000);
    updateStats();

    function updateStats() {
        fetch('/stats')
            .then(response => response.json())
            .then(stats => {
                totalMessagesSpan.textContent = stats.total_messages;
                activeUsersSpan.textContent = stats.active_clients;
            })
            .catch(error => console.error('Error fetching stats:', error));
    }

    setUsernameBtn.addEventListener('click', function() {
        const newUsername = usernameInput.value.trim();
        if (newUsername && newUsername.length >= 2) {
            username = newUsername;
            usernameInput.disabled = true;
            setUsernameBtn.disabled = true;
            connectWebSocket();
        } else {
            alert('Please enter a username with at least 2 characters');
        }
    });

    usernameInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            setUsernameBtn.click();
        }
    });

    function connectWebSocket() {
        if (ws) {
            ws.close();
        }

        connectionStatusSpan.textContent = 'Connecting...';
        connectionStatusSpan.style.color = '#ffc107';

        ws = new WebSocket('ws://' + window.location.host + '/ws?username=' + encodeURIComponent(username));
        
        ws.onopen = function() {
            console.log('WebSocket connection established');
            isConnected = true;
            connectionStatusSpan.textContent = 'Connected';
            connectionStatusSpan.style.color = '#28a745';
            messageInput.disabled = false;
            sendButton.disabled = false;
            messageInput.focus();
        };
        
        ws.onmessage = function(event) {
            const message = JSON.parse(event.data);
            addMessageToChat(message);
            scrollToBottom();
            updateStats();
        };
        
        ws.onclose = function() {
            console.log('WebSocket connection closed');
            isConnected = false;
            connectionStatusSpan.textContent = 'Disconnected';
            connectionStatusSpan.style.color = '#dc3545';
            messageInput.disabled = true;
            sendButton.disabled = true;
            
            // Try to reconnect after 2 seconds
            setTimeout(() => {
                if (username) {
                    connectWebSocket();
                }
            }, 2000);
        };
        
        ws.onerror = function(error) {
            console.error('WebSocket error:', error);
            connectionStatusSpan.textContent = 'Error';
            connectionStatusSpan.style.color = '#dc3545';
        };
    }

    function addMessageToChat(message) {
        const messageElement = document.createElement('div');
        messageElement.className = 'message';
        
        const timestamp = new Date(message.timestamp).toLocaleTimeString();
        
        messageElement.innerHTML = `
            <div class="username">${escapeHtml(message.username)}</div>
            <div class="text">${escapeHtml(message.text)}</div>
            <div class="timestamp">${timestamp}</div>
        `;
        
        chatWindow.appendChild(messageElement);
    }

    function scrollToBottom() {
        chatWindow.scrollTop = chatWindow.scrollHeight;
    }

    messageForm.addEventListener('submit', function(event) {
        event.preventDefault();
        
        const messageText = messageInput.value.trim();
        if (messageText && ws && isConnected) {
            const message = {
                text: messageText
            };
            
            ws.send(JSON.stringify(message));
            messageInput.value = '';
        }
    });

    messageInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            messageForm.dispatchEvent(new Event('submit'));
        }
    });

    // Utility function to escape HTML - FIXED SYNTAX
    function escapeHtml(unsafe) {
        return unsafe
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#039;");
    }

    // Initial focus
    usernameInput.focus();
});