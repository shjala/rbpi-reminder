// PiVoiceReminder Configuration Management
let csrfToken = "";
let autoRefreshInterval = null;

// Get CSRF token from session
async function getCSRFToken() {
    try {
        const response = await fetch("/api/csrf-token");
        if (response.ok) {
            csrfToken = await response.text();
        }
    } catch (error) {
        console.error("Failed to get CSRF token:", error);
    }
}

// Tab switching functionality
function showTab(tabName, event) {
    // Hide all tabs
    document.querySelectorAll(".tab-content").forEach((tab) => {
        tab.classList.remove("active");
    });
    document.querySelectorAll(".nav-btn").forEach((btn) => {
        btn.classList.remove("active");
    });

    // Show selected tab
    document.getElementById(tabName + "-tab").classList.add("active");

    // Find and activate the correct button
    const buttons = document.querySelectorAll(".nav-btn");
    buttons.forEach((btn, index) => {
        if (
            (tabName === "config" && index === 0) ||
            (tabName === "secrets" && index === 1) ||
            (tabName === "logs" && index === 2)
        ) {
            btn.classList.add("active");
        }
    });

    // Load data for the selected tab
    if (tabName === "logs") {
        loadLogs();
    } else if (tabName === "config") {
        loadConfig();
    } else if (tabName === "secrets") {
        loadSecrets();
    }
}

// Load logs function
async function loadLogs() {
    try {
        const response = await fetch("/api/logs");
        const data = await response.text();
        document.getElementById("logs-textarea").value = data;
        // Scroll to bottom to show latest logs
        const textarea = document.getElementById("logs-textarea");
        textarea.scrollTop = textarea.scrollHeight;
    } catch (error) {
        showMessage("logs", "Failed to load logs: " + error.message, "error");
    }
}

// Load configuration data
async function loadConfig() {
    try {
        const response = await fetch("/api/config");
        const data = await response.text();
        document.getElementById("config-textarea").value = data;
    } catch (error) {
        showMessage(
            "config",
            "Failed to load configuration: " + error.message,
            "error",
        );
    }
}

async function loadSecrets() {
    try {
        const response = await fetch("/api/secrets");
        const data = await response.text();
        document.getElementById("secrets-textarea").value = data;
    } catch (error) {
        showMessage(
            "secrets",
            "Failed to load secrets: " + error.message,
            "error",
        );
    }
}

// Save configuration
async function saveConfig() {
    const configData = document.getElementById("config-textarea").value;
    try {
        const response = await fetch("/api/config/save", {
            method: "POST",
            headers: {
                "Content-Type": "text/plain",
                "X-CSRF-Token": csrfToken,
            },
            body: configData,
        });

        if (response.ok) {
            showMessage(
                "config",
                "Configuration saved successfully!",
                "success",
            );
        } else {
            const error = await response.text();
            showMessage(
                "config",
                "Failed to save configuration: " + error,
                "error",
            );
        }
    } catch (error) {
        showMessage(
            "config",
            "Failed to save configuration: " + error.message,
            "error",
        );
    }
}

async function saveSecrets() {
    const secretsData = document.getElementById("secrets-textarea").value;
    try {
        const response = await fetch("/api/secrets/save", {
            method: "POST",
            headers: {
                "Content-Type": "text/plain",
                "X-CSRF-Token": csrfToken,
            },
            body: secretsData,
        });

        if (response.ok) {
            showMessage("secrets", "Secrets saved successfully!", "success");
        } else {
            const error = await response.text();
            showMessage("secrets", "Failed to save secrets: " + error, "error");
        }
    } catch (error) {
        showMessage(
            "secrets",
            "Failed to save secrets: " + error.message,
            "error",
        );
    }
}

async function clearLogs() {
    if (
        !confirm(
            "Are you sure you want to clear all logs? This action cannot be undone.",
        )
    ) {
        return;
    }

    try {
        const response = await fetch("/api/logs/clear", {
            method: "POST",
            headers: {
                "X-CSRF-Token": csrfToken,
            },
        });

        if (response.ok) {
            document.getElementById("logs-textarea").value = "";
            showMessage("logs", "Logs cleared successfully!", "success");
        } else {
            const error = await response.text();
            showMessage("logs", "Failed to clear logs: " + error, "error");
        }
    } catch (error) {
        showMessage("logs", "Failed to clear logs: " + error.message, "error");
    }
}

function toggleAutoRefresh() {
    const checkbox = document.getElementById("auto-refresh");

    if (checkbox.checked) {
        // Start auto-refresh every 30 seconds
        autoRefreshInterval = setInterval(() => {
            // Only refresh if logs tab is active
            if (
                document.getElementById("logs-tab").classList.contains("active")
            ) {
                loadLogs();
            }
        }, 30000);
        showMessage("logs", "Auto-refresh enabled (30 seconds)", "success");
    } else {
        // Stop auto-refresh
        if (autoRefreshInterval) {
            clearInterval(autoRefreshInterval);
            autoRefreshInterval = null;
        }
        showMessage("logs", "Auto-refresh disabled", "success");
    }
}

function showMessage(type, message, level) {
    const messageDiv = document.getElementById(type + "-message");
    messageDiv.textContent = message;
    messageDiv.className = "message " + level;
    messageDiv.style.display = "block";

    // Hide message after 5 seconds
    setTimeout(() => {
        messageDiv.style.display = "none";
    }, 5000);
}

// Initialize application when page loads
window.onload = async function () {
    await getCSRFToken();
    loadConfig();
    loadSecrets();
    loadLogs();
};

// Clean up auto-refresh interval when page is unloaded
window.onbeforeunload = function () {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
    }
};
