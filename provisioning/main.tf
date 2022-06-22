terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
      version = "3.5.0"
    }
  }
}

provider "google" {
  #credentials = file(var.credentials_file)
  project = var.project
  region  = "asia-northeast1"
  zone    = "asia-northeast1-c"
}
resource "google_project_service" "enable_api" {
  service = "compute.googleapis.com"
}

resource "google_compute_network" "vpc_network" {
  name = "my-network"
}

resource "google_compute_instance" "vm_instance" {
  name         = "my-instance"
  machine_type = "n1-standard-1"
  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-9"
    }
  }
  network_interface {
    network = google_compute_network.vpc_network.self_link
    access_config {
      nat_ip = google_compute_address.vm_static_ip.address
    }
  }
}


resource "google_compute_address" "vm_static_ip" {
  name = "static-ip"
}

  

