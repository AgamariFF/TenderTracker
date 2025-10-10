// Функции для выбора/очистки всех чекбоксов
function selectAll(className) {
    document.querySelectorAll('.' + className).forEach(checkbox => {
        checkbox.checked = true;
    });
}

function deselectAll(className) {
    document.querySelectorAll('.' + className).forEach(checkbox => {
        checkbox.checked = false;
    });
}

document.getElementById('searchForm').addEventListener('submit', function(e) {
    e.preventDefault();
    
    // Показываем индикатор загрузки
    document.getElementById('loadingSection').style.display = 'block';
    document.getElementById('resultSection').style.display = 'none';
    document.getElementById('errorSection').style.display = 'none';

    // Собираем данные формы
    const formData = new FormData(this);
    
    // Добавляем выбранные федеральные округа (customerPlace)
    const customerPlace = Array.from(document.querySelectorAll('.customer-place:checked'))
        .map(checkbox => checkbox.value);
    
    customerPlace.forEach(value => {
        formData.append('vent_customer_place', value);
    });

    // Добавляем выбранные коды регионов (delKladrIds)
    const delKladrIds = Array.from(document.querySelectorAll('.kladr-ids:checked'))
        .map(checkbox => checkbox.value);
    
    delKladrIds.forEach(value => {
        formData.append('vent_del_kladr_ids', value);
    });

    // Отправляем запрос
    fetch('/tender/searchTenders', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        document.getElementById('loadingSection').style.display = 'none';
        
        if (data.error) {
            showError(data.error);
        } else {
            showSuccess(data);
        }
    })
    .catch(error => {
        document.getElementById('loadingSection').style.display = 'none';
        showError('Ошибка сети: ' + error.message);
    });
});

function showSuccess(data) {
    document.getElementById('resultSection').style.display = 'block';
    
    const statsElement = document.getElementById('searchStats');
    
    if (data.stats && data.stats.totalFound !== undefined) {
        statsElement.style.display = 'block';
        statsElement.innerHTML = `
            <div class="row">
                <div class="col-md-4">
                    <div class="card bg-light">
                        <div class="card-body text-center">
                            <h4 class="text-primary">${data.stats.totalFound}</h4>
                            <small class="text-muted">Найдено закупок</small>
                        </div>
                    </div>
                </div>
            </div>
        `;
    } else {
        statsElement.style.display = 'none';
        statsElement.innerHTML = '';
    }
}

function showError(message) {
    document.getElementById('errorSection').style.display = 'block';
    document.getElementById('errorMessage').textContent = message;
}

function downloadFile() {
    // Создаем временную ссылку для скачивания
    const link = document.createElement('a');
    link.href = '/tender/download?filename=Закупки.xlsx&t=' + new Date().getTime();
    link.download = 'Закупки_' + new Date().toISOString().split('T')[0] + '.xlsx';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    // Уже есть кнопки в HTML, поэтому ничего дополнительного не нужно
});


// Функция показа помощи
function showHelp() {
    const helpModal = new bootstrap.Modal(document.getElementById('helpModal'));
    helpModal.show();
}

function changePort(port, path = '') {
    const host = window.location.hostname;
    window.location.href = `//${host}:${port}/${path}`;
}